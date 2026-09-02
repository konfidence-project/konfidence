package stage_test

import (
	"context"
	"testing"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	landscapedomain "github.com/konfidence-project/konfidence/internal/landscape"
	"github.com/konfidence-project/konfidence/internal/stage"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(s)
	_ = konfidence.AddToScheme(s)
	return s
}

func fakeClient(objs ...client.Object) client.Client {
	return fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(objs...).Build()
}

// countingReader records the namespaces a repository lists from, so tests can pin
// the efficiency contract: one Stage LIST and one StageVersion LIST per scope entry.
type countingReader struct {
	client.Reader
	listsByNamespace map[string]int
}

func newCountingReader(reader client.Reader) *countingReader {
	return &countingReader{Reader: reader, listsByNamespace: map[string]int{}}
}

func (r *countingReader) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	options := &client.ListOptions{}
	for _, opt := range opts {
		opt.ApplyToList(options)
	}
	r.listsByNamespace[options.Namespace]++
	return r.Reader.List(ctx, list, opts...)
}

// stageFixture builds a stage. The fake client does not maintain metadata.generation,
// so it is set explicitly.
func stageFixture(name, namespace, vector string, generation int64) *konfidence.Stage {
	return &konfidence.Stage{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Generation: generation},
		Spec:       konfidence.StageSpec{Vector: vector},
	}
}

func withActiveStageVersion(s *konfidence.Stage, name string) *konfidence.Stage {
	s.Status.ActiveStageVersion = &konfidence.StageVersionReference{Name: name}
	return s
}

func stageVersionFixture(name, namespace, stageName, vector string, stageGeneration int64) *konfidence.StageVersion {
	return &konfidence.StageVersion{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: konfidence.StageVersionSpec{
			Vector:          vector,
			StageGeneration: stageGeneration,
			StageRef:        &konfidence.StageReference{Name: stageName},
		},
	}
}

func scopeFixture(landscapeName, managedNamespace string) landscapedomain.ScopedLandscape {
	return landscapedomain.ScopedLandscape{
		Landscape: konfidence.Landscape{
			ObjectMeta: metav1.ObjectMeta{Name: landscapeName, Namespace: "kden-p-my-project"},
			Status:     konfidence.LandscapeStatus{Namespace: managedNamespace},
		},
		Namespace: managedNamespace,
	}
}

func TestListForScope_AggregatesLandscapes(t *testing.T) {
	devStage := stageFixture("app", "kden-l-dev", "vector:1", 1)
	stagingStage := stageFixture("app", "kden-l-staging", "vector:1", 1)
	repo := stage.NewRepository(fakeClient(devStage, stagingStage))

	resolved, err := repo.ListForScope(context.Background(), []landscapedomain.ScopedLandscape{
		scopeFixture("dev", "kden-l-dev"),
		scopeFixture("staging", "kden-l-staging"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(resolved))
	}
	if resolved[0].LandscapeID != "dev" {
		t.Errorf("expected LandscapeID %q, got %q", "dev", resolved[0].LandscapeID)
	}
	if resolved[1].LandscapeID != "staging" {
		t.Errorf("expected LandscapeID %q, got %q", "staging", resolved[1].LandscapeID)
	}
}

func TestListForScope_ReadsOnlyScopedNamespaces(t *testing.T) {
	devStage := stageFixture("app", "kden-l-dev", "vector:1", 1)
	foreignStage := stageFixture("app", "kden-l-other-project", "vector:1", 1)
	reader := newCountingReader(fakeClient(devStage, foreignStage))
	repo := stage.NewRepository(reader)

	resolved, err := repo.ListForScope(context.Background(), []landscapedomain.ScopedLandscape{
		scopeFixture("dev", "kden-l-dev"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 stage, got %d", len(resolved))
	}
	if resolved[0].Stage.Namespace != "kden-l-dev" {
		t.Errorf("expected namespace %q, got %q", "kden-l-dev", resolved[0].Stage.Namespace)
	}
	if got := reader.listsByNamespace["kden-l-dev"]; got != 2 {
		t.Errorf("expected 2 lists in namespace %q, got %d", "kden-l-dev", got)
	}
	if len(reader.listsByNamespace) != 1 {
		t.Errorf("expected lists in exactly 1 namespace, got %v", reader.listsByNamespace)
	}
}

func TestListForScope_ResolvesTargetByGenerationAndVector(t *testing.T) {
	s := stageFixture("app", "kden-l-dev", "vector:2", 2)
	previous := stageVersionFixture("app-abc", "kden-l-dev", "app", "vector:1", 1)
	current := stageVersionFixture("app-def", "kden-l-dev", "app", "vector:2", 2)
	repo := stage.NewRepository(fakeClient(s, previous, current))

	resolved, err := repo.ListForScope(context.Background(), []landscapedomain.ScopedLandscape{
		scopeFixture("dev", "kden-l-dev"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 stage, got %d", len(resolved))
	}
	if resolved[0].Target == nil {
		t.Fatalf("expected a target stage version")
	}
	if resolved[0].Target.Name != "app-def" {
		t.Errorf("expected target %q, got %q", "app-def", resolved[0].Target.Name)
	}
	if resolved[0].Active != nil {
		t.Errorf("expected no active stage version, got %q", resolved[0].Active.Name)
	}
}

func TestListForScope_TargetRequiresGenerationAndVectorToMatch(t *testing.T) {
	s := stageFixture("app", "kden-l-dev", "vector:2", 2)
	// same generation as the stage, but the vector of an earlier spec
	sameGeneration := stageVersionFixture("app-gen", "kden-l-dev", "app", "vector:1", 2)
	// the stage's vector, but created for an earlier generation
	sameVector := stageVersionFixture("app-vec", "kden-l-dev", "app", "vector:2", 1)
	repo := stage.NewRepository(fakeClient(s, sameGeneration, sameVector))

	resolved, err := repo.ListForScope(context.Background(), []landscapedomain.ScopedLandscape{
		scopeFixture("dev", "kden-l-dev"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 stage, got %d", len(resolved))
	}
	if resolved[0].Target != nil {
		t.Errorf("expected no target stage version, got %q", resolved[0].Target.Name)
	}
}

func TestListForScope_TargetIsTheFullMatchAmongPartialMatches(t *testing.T) {
	s := stageFixture("app", "kden-l-dev", "vector:2", 2)
	sameGeneration := stageVersionFixture("app-gen", "kden-l-dev", "app", "vector:1", 2)
	sameVector := stageVersionFixture("app-vec", "kden-l-dev", "app", "vector:2", 1)
	target := stageVersionFixture("app-target", "kden-l-dev", "app", "vector:2", 2)
	repo := stage.NewRepository(fakeClient(s, sameGeneration, sameVector, target))

	resolved, err := repo.ListForScope(context.Background(), []landscapedomain.ScopedLandscape{
		scopeFixture("dev", "kden-l-dev"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved[0].Target == nil {
		t.Fatalf("expected a target stage version")
	}
	if resolved[0].Target.Name != "app-target" {
		t.Errorf("expected target %q, got %q", "app-target", resolved[0].Target.Name)
	}
}

func TestListForScope_ResolvesActiveWithoutTarget(t *testing.T) {
	s := withActiveStageVersion(stageFixture("app", "kden-l-dev", "vector:2", 2), "app-abc")
	previous := stageVersionFixture("app-abc", "kden-l-dev", "app", "vector:1", 1)
	repo := stage.NewRepository(fakeClient(s, previous))

	resolved, err := repo.ListForScope(context.Background(), []landscapedomain.ScopedLandscape{
		scopeFixture("dev", "kden-l-dev"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved[0].Target != nil {
		t.Errorf("expected no target stage version, got %q", resolved[0].Target.Name)
	}
	if resolved[0].Active == nil {
		t.Fatalf("expected an active stage version")
	}
	if resolved[0].Active.Name != "app-abc" {
		t.Errorf("expected active %q, got %q", "app-abc", resolved[0].Active.Name)
	}
}

func TestListForScope_FreshStageHasNoVersions(t *testing.T) {
	s := stageFixture("app", "kden-l-dev", "vector:1", 1)
	repo := stage.NewRepository(fakeClient(s))

	resolved, err := repo.ListForScope(context.Background(), []landscapedomain.ScopedLandscape{
		scopeFixture("dev", "kden-l-dev"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 stage, got %d", len(resolved))
	}
	if resolved[0].Target != nil || resolved[0].Active != nil {
		t.Errorf("expected no versions, got target %v and active %v", resolved[0].Target, resolved[0].Active)
	}
}

func TestListForScope_DanglingActiveRefResolvesToNil(t *testing.T) {
	s := withActiveStageVersion(stageFixture("app", "kden-l-dev", "vector:1", 1), "app-gone")
	current := stageVersionFixture("app-abc", "kden-l-dev", "app", "vector:1", 1)
	repo := stage.NewRepository(fakeClient(s, current))

	resolved, err := repo.ListForScope(context.Background(), []landscapedomain.ScopedLandscape{
		scopeFixture("dev", "kden-l-dev"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved[0].Active != nil {
		t.Errorf("expected no active stage version, got %q", resolved[0].Active.Name)
	}
	if resolved[0].Target == nil {
		t.Errorf("expected a target stage version")
	}
}

func TestListForScope_SkipsProvisioningLandscape(t *testing.T) {
	devStage := stageFixture("app", "kden-l-dev", "vector:1", 1)
	reader := newCountingReader(fakeClient(devStage))
	repo := stage.NewRepository(reader)

	resolved, err := repo.ListForScope(context.Background(), []landscapedomain.ScopedLandscape{
		scopeFixture("provisioning", ""),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 0 {
		t.Errorf("expected 0 stages, got %d", len(resolved))
	}
	if len(reader.listsByNamespace) != 0 {
		t.Errorf("expected no lists, got %v", reader.listsByNamespace)
	}
}

func TestListForScope_IgnoresVersionWithoutStageRef(t *testing.T) {
	s := stageFixture("app", "kden-l-dev", "vector:1", 1)
	orphan := stageVersionFixture("orphan", "kden-l-dev", "app", "vector:1", 1)
	orphan.Spec.StageRef = nil
	repo := stage.NewRepository(fakeClient(s, orphan))

	resolved, err := repo.ListForScope(context.Background(), []landscapedomain.ScopedLandscape{
		scopeFixture("dev", "kden-l-dev"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved[0].Target != nil {
		t.Errorf("expected no target stage version, got %q", resolved[0].Target.Name)
	}
}

func TestListForScope_GroupsVersionsPerStage(t *testing.T) {
	repo := stage.NewRepository(fakeClient(
		stageFixture("api", "kden-l-dev", "vector:1", 1),
		stageFixture("web", "kden-l-dev", "vector:1", 1),
		stageVersionFixture("api-abc", "kden-l-dev", "api", "vector:1", 1),
		stageVersionFixture("web-def", "kden-l-dev", "web", "vector:1", 1),
	))

	resolved, err := repo.ListForScope(context.Background(), []landscapedomain.ScopedLandscape{
		scopeFixture("dev", "kden-l-dev"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(resolved))
	}
	targets := map[string]string{}
	for _, rs := range resolved {
		if rs.Target == nil {
			t.Fatalf("expected a target stage version for stage %q", rs.Stage.Name)
		}
		targets[rs.Stage.Name] = rs.Target.Name
	}
	if targets["api"] != "api-abc" {
		t.Errorf("expected target %q for stage %q, got %q", "api-abc", "api", targets["api"])
	}
	if targets["web"] != "web-def" {
		t.Errorf("expected target %q for stage %q, got %q", "web-def", "web", targets["web"])
	}
}

func TestListForScope_SortsByLandscapeAndName(t *testing.T) {
	repo := stage.NewRepository(fakeClient(
		stageFixture("web", "kden-l-staging", "vector:1", 1),
		stageFixture("api", "kden-l-staging", "vector:1", 1),
		stageFixture("web", "kden-l-dev", "vector:1", 1),
		stageFixture("api", "kden-l-dev", "vector:1", 1),
	))

	// the scope is passed in reverse order, so the expected order can only come from the sort
	scope := []landscapedomain.ScopedLandscape{
		scopeFixture("staging", "kden-l-staging"),
		scopeFixture("dev", "kden-l-dev"),
	}
	expected := []struct{ landscape, name string }{
		{"dev", "api"},
		{"dev", "web"},
		{"staging", "api"},
		{"staging", "web"},
	}

	// ordering is part of the response contract, so it has to be stable across calls
	for call := range 2 {
		resolved, err := repo.ListForScope(context.Background(), scope)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resolved) != len(expected) {
			t.Fatalf("expected %d stages, got %d", len(expected), len(resolved))
		}
		for i, e := range expected {
			if resolved[i].LandscapeID != e.landscape || resolved[i].Stage.Name != e.name {
				t.Errorf("call %d: expected (%q, %q) at index %d, got (%q, %q)",
					call, e.landscape, e.name, i, resolved[i].LandscapeID, resolved[i].Stage.Name)
			}
		}
	}
}

func TestListForScope_DoesNotLeakForeignNamespaces(t *testing.T) {
	own := stageFixture("app", "kden-l-dev", "vector:1", 1)
	foreign := stageFixture("secret", "kden-l-other-project", "vector:1", 1)
	foreignVersion := stageVersionFixture("app-abc", "kden-l-other-project", "app", "vector:1", 1)
	repo := stage.NewRepository(fakeClient(own, foreign, foreignVersion))

	resolved, err := repo.ListForScope(context.Background(), []landscapedomain.ScopedLandscape{
		scopeFixture("dev", "kden-l-dev"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 stage, got %d", len(resolved))
	}
	if resolved[0].Stage.Name != "app" {
		t.Fatalf("foreign stage leaked into the result: %q", resolved[0].Stage.Name)
	}
	if resolved[0].Target != nil {
		t.Errorf("foreign stage version leaked into the result: %q", resolved[0].Target.Name)
	}
}
