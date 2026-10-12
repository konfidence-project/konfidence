package stage_test

import (
	"context"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	landscapedomain "github.com/konfidence-project/konfidence/internal/landscape"
	"github.com/konfidence-project/konfidence/internal/stage"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newScheme() *runtime.Scheme {
	GinkgoHelper()
	scheme := runtime.NewScheme()
	Expect(clientgoscheme.AddToScheme(scheme)).To(Succeed())
	Expect(konfidence.AddToScheme(scheme)).To(Succeed())
	return scheme
}

func fakeClient(objs ...client.Object) client.Client {
	GinkgoHelper()
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

func listForDevScope(objects ...client.Object) []stage.ResolvedStage {
	GinkgoHelper()
	resolved, err := stage.NewRepository(fakeClient(objects...)).ListForScope(
		context.Background(),
		[]landscapedomain.ScopedLandscape{scopeFixture("dev", "kden-l-dev")},
	)
	Expect(err).NotTo(HaveOccurred())
	return resolved
}

var _ = Describe("Repository ListForScope", func() {
	It("aggregates landscapes", func() {
		repository := stage.NewRepository(fakeClient(
			stageFixture("app", "kden-l-dev", "vector:1", 1),
			stageFixture("app", "kden-l-staging", "vector:1", 1),
		))
		resolved, err := repository.ListForScope(context.Background(), []landscapedomain.ScopedLandscape{
			scopeFixture("dev", "kden-l-dev"),
			scopeFixture("staging", "kden-l-staging"),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(resolved).To(ConsistOf(
			HaveField("LandscapeID", "dev"),
			HaveField("LandscapeID", "staging"),
		))
	})

	It("reads only scoped namespaces", func() {
		reader := newCountingReader(fakeClient(
			stageFixture("app", "kden-l-dev", "vector:1", 1),
			stageFixture("app", "kden-l-other-project", "vector:1", 1),
		))
		resolved, err := stage.NewRepository(reader).ListForScope(context.Background(), []landscapedomain.ScopedLandscape{
			scopeFixture("dev", "kden-l-dev"),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(resolved).To(ConsistOf(HaveField("Stage.Namespace", "kden-l-dev")))
		Expect(reader.listsByNamespace).To(Equal(map[string]int{"kden-l-dev": 2}))
	})

	It("resolves the target by generation and vector", func() {
		resolved := listForDevScope(
			stageFixture("app", "kden-l-dev", "vector:2", 2),
			stageVersionFixture("app-abc", "kden-l-dev", "app", "vector:1", 1),
			stageVersionFixture("app-def", "kden-l-dev", "app", "vector:2", 2),
		)
		Expect(resolved).To(ConsistOf(And(
			HaveField("Target.Name", "app-def"),
			HaveField("Active", BeNil()),
		)))
	})

	It("requires generation and vector to match the target", func() {
		resolved := listForDevScope(
			stageFixture("app", "kden-l-dev", "vector:2", 2),
			stageVersionFixture("app-gen", "kden-l-dev", "app", "vector:1", 2),
			stageVersionFixture("app-vec", "kden-l-dev", "app", "vector:2", 1),
		)
		Expect(resolved).To(ConsistOf(HaveField("Target", BeNil())))
	})

	It("selects the full target match among partial matches", func() {
		resolved := listForDevScope(
			stageFixture("app", "kden-l-dev", "vector:2", 2),
			stageVersionFixture("app-gen", "kden-l-dev", "app", "vector:1", 2),
			stageVersionFixture("app-vec", "kden-l-dev", "app", "vector:2", 1),
			stageVersionFixture("app-target", "kden-l-dev", "app", "vector:2", 2),
		)
		Expect(resolved).To(ConsistOf(HaveField("Target.Name", "app-target")))
	})

	It("resolves an active version without a target", func() {
		resolved := listForDevScope(
			withActiveStageVersion(stageFixture("app", "kden-l-dev", "vector:2", 2), "app-abc"),
			stageVersionFixture("app-abc", "kden-l-dev", "app", "vector:1", 1),
		)
		Expect(resolved).To(ConsistOf(And(
			HaveField("Target", BeNil()),
			HaveField("Active.Name", "app-abc"),
		)))
	})

	It("returns no versions for a fresh stage", func() {
		resolved := listForDevScope(stageFixture("app", "kden-l-dev", "vector:1", 1))
		Expect(resolved).To(ConsistOf(And(
			HaveField("Target", BeNil()),
			HaveField("Active", BeNil()),
		)))
	})

	It("resolves a dangling active reference to nil", func() {
		resolved := listForDevScope(
			withActiveStageVersion(stageFixture("app", "kden-l-dev", "vector:1", 1), "app-gone"),
			stageVersionFixture("app-abc", "kden-l-dev", "app", "vector:1", 1),
		)
		Expect(resolved).To(ConsistOf(And(
			HaveField("Active", BeNil()),
			HaveField("Target", Not(BeNil())),
		)))
	})

	It("skips a provisioning landscape", func() {
		reader := newCountingReader(fakeClient(stageFixture("app", "kden-l-dev", "vector:1", 1)))
		resolved, err := stage.NewRepository(reader).ListForScope(context.Background(), []landscapedomain.ScopedLandscape{
			scopeFixture("provisioning", ""),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(resolved).To(BeEmpty())
		Expect(reader.listsByNamespace).To(BeEmpty())
	})

	It("ignores a version without a stage reference", func() {
		orphan := stageVersionFixture("orphan", "kden-l-dev", "app", "vector:1", 1)
		orphan.Spec.StageRef = nil
		resolved := listForDevScope(stageFixture("app", "kden-l-dev", "vector:1", 1), orphan)
		Expect(resolved).To(ConsistOf(HaveField("Target", BeNil())))
	})

	It("groups versions per stage", func() {
		resolved := listForDevScope(
			stageFixture("api", "kden-l-dev", "vector:1", 1),
			stageFixture("web", "kden-l-dev", "vector:1", 1),
			stageVersionFixture("api-abc", "kden-l-dev", "api", "vector:1", 1),
			stageVersionFixture("web-def", "kden-l-dev", "web", "vector:1", 1),
		)
		Expect(resolved).To(ConsistOf(
			And(HaveField("Stage.Name", "api"), HaveField("Target.Name", "api-abc")),
			And(HaveField("Stage.Name", "web"), HaveField("Target.Name", "web-def")),
		))
	})

	It("sorts by landscape and name on every call", func() {
		repository := stage.NewRepository(fakeClient(
			stageFixture("web", "kden-l-staging", "vector:1", 1),
			stageFixture("api", "kden-l-staging", "vector:1", 1),
			stageFixture("web", "kden-l-dev", "vector:1", 1),
			stageFixture("api", "kden-l-dev", "vector:1", 1),
		))
		scope := []landscapedomain.ScopedLandscape{
			scopeFixture("staging", "kden-l-staging"),
			scopeFixture("dev", "kden-l-dev"),
		}
		expected := []string{"dev/api", "dev/web", "staging/api", "staging/web"}

		for range 2 {
			resolved, err := repository.ListForScope(context.Background(), scope)
			Expect(err).NotTo(HaveOccurred())
			actual := make([]string, 0, len(resolved))
			for _, item := range resolved {
				actual = append(actual, item.LandscapeID+"/"+item.Stage.Name)
			}
			Expect(actual).To(Equal(expected))
		}
	})

	It("does not leak foreign namespaces", func() {
		resolved := listForDevScope(
			stageFixture("app", "kden-l-dev", "vector:1", 1),
			stageFixture("secret", "kden-l-other-project", "vector:1", 1),
			stageVersionFixture("app-abc", "kden-l-other-project", "app", "vector:1", 1),
		)
		Expect(resolved).To(ConsistOf(And(
			HaveField("Stage.Name", "app"),
			HaveField("Target", BeNil()),
		)))
	})
})
