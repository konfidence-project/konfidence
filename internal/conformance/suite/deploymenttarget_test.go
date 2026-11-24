package suite

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/conformance/setup"
	"github.com/onsi/ginkgo/v2/reporters"
	"github.com/onsi/ginkgo/v2/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const targetWithoutReference = "no-reference"

// Each fixture executes the actual scenarios using the existing framework.
// Ginkgo permits one invocation per process, so parent tests launch subprocesses.
func TestDeploymentTargetFixture(t *testing.T) {
	fixture := os.Getenv("KDEN_CONFORMANCE_TARGET_FIXTURE")
	if fixture == "" {
		t.Skip("subprocess fixture")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	inputs := targetFixtureInputs(fixture)
	originalSpec := inputs.DeploymentTarget.DeepCopy()
	originalSetup := inputs.TargetSetup[0].DeepCopy()
	c := targetFixtureClient(t, fixture, inputs, cancel)
	environment := &Environment{Context: ctx, Inputs: inputs, Client: c}
	result, runErr := runFramework(environment, Options{ReportDir: os.Getenv("KDEN_CONFORMANCE_FIXTURE_REPORT_DIR")},
		func(env *Environment) {
			registerLifecycle(env)
			registerScenarios(env)
		})
	if !reflect.DeepEqual(originalSpec, &inputs.DeploymentTarget) || !reflect.DeepEqual(originalSetup, &inputs.TargetSetup[0]) {
		t.Error("scenarios mutated caller-owned inputs")
	}
	if fixture != "cleanup-failure" {
		namespaces := &corev1.NamespaceList{}
		if err := c.List(context.Background(), namespaces); err != nil || len(namespaces.Items) != 1 ||
			namespaces.Items[0].Name != "developer-owned" {
			t.Errorf("test namespaces were not cleaned up or developer namespace was affected: %v", err)
		}
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(result.ReportDir, "result.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	if runErr != nil {
		t.Fatal(runErr)
	}
}

func targetFixtureInputs(fixture string) *setup.Inputs {
	inputs := &setup.Inputs{
		DeploymentTarget: konfidence.DeploymentTargetSpec{
			ArtifactType: konfidence.MustArtifactType("app.example.com"),
			Parameters:   &konfidence.Parameters{Kind: "Secret", Name: "credentials"},
		},
		TargetSetup: []unstructured.Unstructured{{Object: map[string]any{
			"apiVersion": "v1", "kind": "Secret",
			"metadata":   map[string]any{"name": "credentials"},
			"stringData": map[string]any{"token": "sensitive-target-value"},
		}}},
	}
	if fixture == targetWithoutReference {
		inputs.DeploymentTarget.Parameters = nil
	}
	if fixture == "cluster-scoped-setup" {
		inputs.TargetSetup = []unstructured.Unstructured{{Object: map[string]any{
			"apiVersion": "v1", "kind": "Namespace",
			"metadata": map[string]any{"name": "developer-owned"},
		}}}
	}
	if fixture == "registry" {
		inputs.RegistrySecrets = []*corev1.Secret{{
			ObjectMeta: metav1.ObjectMeta{Name: "registry"},
			Data:       map[string][]byte{".dockerconfigjson": []byte(`{"auths":{}}`)},
		}}
	}
	return inputs
}

func targetFixtureClient(t *testing.T, fixture string, inputs *setup.Inputs, cancel context.CancelFunc) client.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := konfidence.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	mapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{corev1.SchemeGroupVersion})
	mapper.Add(corev1.SchemeGroupVersion.WithKind("Secret"), meta.RESTScopeNamespace)
	mapper.Add(corev1.SchemeGroupVersion.WithKind("Namespace"), meta.RESTScopeRoot)
	foreign := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "developer-owned"}}
	reads := map[client.ObjectKey]int{}
	c := fake.NewClientBuilder().WithScheme(scheme).WithRESTMapper(mapper).WithObjects(foreign).
		WithStatusSubresource(&konfidence.DeploymentTarget{}).
		WithInterceptorFuncs(interceptor.Funcs{
			Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				if _, ok := obj.(*unstructured.Unstructured); ok && fixture == "setup-failure" {
					return errors.New("sensitive-target-value")
				}
				target, isTarget := obj.(*konfidence.DeploymentTarget)
				if isTarget && fixture == "create-failure" {
					return errors.New("sensitive-target-value")
				}
				if isTarget && fixture == "cancellation" {
					cancel()
					return context.Canceled
				}
				if err := c.Create(ctx, obj, opts...); err != nil {
					return err
				}
				if !isTarget {
					return nil
				}
				// Simulate controller writes without mutating Create's response.
				return writeTargetFixtureStatus(ctx, c, target.DeepCopy(), inputs.DeploymentTarget.ArtifactType, fixture)
			},
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if err := c.Get(ctx, key, obj, opts...); err != nil {
					return err
				}
				target, ok := obj.(*konfidence.DeploymentTarget)
				if ok && strings.HasPrefix(target.Name, "foreign-") {
					reads[key]++
					if fixture == "foreign-metadata" {
						target.Finalizers = []string{"example.com/unexpected"}
					}
					if fixture == "delayed-foreign-status" && reads[key] > 1 {
						target.Status.Controller = "sensitive-target-value"
					}
				}
				return nil
			},
			Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
				if ctx.Err() != nil {
					t.Error("cleanup received a canceled context")
				}
				if fixture == "cleanup-failure" {
					return errors.New("sensitive-target-value")
				}
				return c.Delete(ctx, obj, opts...)
			},
		}).Build()
	return c
}

func writeTargetFixtureStatus(
	ctx context.Context, c client.Client, target *konfidence.DeploymentTarget, supported konfidence.ArtifactType, fixture string,
) error {
	if target.Spec.ArtifactType != supported {
		if (fixture == "foreign-type-status" && target.Name == "foreign-0") ||
			(fixture == "foreign-provider-status" && target.Name == "foreign-1") {
			target.Status.Controller = "sensitive-target-value"
		}
		return c.Status().Update(ctx, target)
	}
	status, reason, message := metav1.ConditionTrue, "Accepted", ""
	missing := false
	if parameters := target.Spec.Parameters; parameters != nil {
		secret := &corev1.Secret{}
		missing = c.Get(ctx, client.ObjectKey{Name: parameters.Name, Namespace: target.Namespace}, secret) != nil
	}
	if missing {
		status, reason, message = metav1.ConditionFalse, "InvalidParameters", "Referenced resource not found"
	}
	if fixture == "invalid-accepted" && missing {
		status = metav1.ConditionTrue
	}
	if fixture == "missing-reason" && missing {
		reason = ""
	}
	if fixture == "missing-message" && missing {
		message = " "
	}
	target.Status.Conditions = []metav1.Condition{{
		Type: konfidence.DeploymentTargetReadyCondition, Status: status,
		Reason: reason, Message: message, LastTransitionTime: metav1.Now(),
	}}
	if fixture == "never-ready" && !missing {
		target.Status.Conditions = nil
	}
	return c.Status().Update(ctx, target)
}

func TestDeploymentTargetReports(t *testing.T) {
	for _, fixture := range []string{
		"success", "registry", targetWithoutReference, "foreign-type-status", "foreign-provider-status",
		"foreign-metadata", "delayed-foreign-status", "invalid-accepted", "missing-reason", "missing-message",
		"never-ready", "setup-failure", "create-failure", "cancellation", "cleanup-failure", "cluster-scoped-setup",
	} {
		t.Run(fixture, func(t *testing.T) {
			directory := t.TempDir()
			command := exec.Command(os.Args[0], "-test.run=^TestDeploymentTargetFixture$")
			command.Env = append(os.Environ(),
				"KDEN_CONFORMANCE_TARGET_FIXTURE="+fixture, "KDEN_CONFORMANCE_FIXTURE_REPORT_DIR="+directory,
				"KDEN_CONFORMANCE_TIMEOUT=1s", "KDEN_CONFORMANCE_OBSERVATION_TIMEOUT=300ms",
			)
			output, err := command.CombinedOutput()
			shouldFail := fixture != "success" && fixture != "registry" && fixture != targetWithoutReference
			if (err != nil) != shouldFail {
				t.Fatalf("unexpected subprocess outcome: %v\n%s", err, output)
			}
			var reports []Report
			readJSON(t, filepath.Join(directory, "report.json"), &reports)
			if len(reports) != 1 || reports[0].SuiteSucceeded == shouldFail {
				t.Fatalf("unexpected framework status: %+v", reports)
			}
			specs := reports[0].SpecReports.WithLeafNodeType(types.NodeTypeIt)
			states := map[string]types.SpecState{}
			for _, spec := range specs {
				if len(spec.LeafNodeLabels) != 1 {
					t.Fatalf("expected one keyword label per spec, got %v", spec.LeafNodeLabels)
				}
				keyword := spec.LeafNodeLabels[0]
				if _, duplicate := states[keyword]; duplicate {
					t.Fatalf("duplicate keyword spec: %s", keyword)
				}
				states[keyword] = spec.State
			}
			for _, keyword := range []string{"0018", "0019", "0022", "0027", "0029"} {
				if _, present := states[keyword]; !present {
					t.Errorf("missing keyword spec: %s", keyword)
				}
			}
			if len(states) != 5 {
				t.Fatalf("expected exactly five keyword specs: %v", states)
			}
			expectedFailures := map[string][]string{
				"foreign-type-status":     {"0018", "0019"},
				"foreign-provider-status": {"0018", "0019"},
				"foreign-metadata":        {"0018"},
				"delayed-foreign-status":  {"0018", "0019"},
				"invalid-accepted":        {"0022", "0029"},
				"missing-reason":          {"0029"},
				"missing-message":         {"0029"},
				"never-ready":             {"0018", "0019", "0027"},
			}
			for _, keyword := range expectedFailures[fixture] {
				if !states[keyword].Is(types.SpecStateFailureStates) {
					t.Errorf("[%s] violation was not detected: %s", keyword, states[keyword])
				}
			}
			if fixture == "foreign-metadata" && states["0019"] != types.SpecStatePassed {
				t.Error("[0019] must only check status, not metadata")
			}
			if fixture == "never-ready" && states["0027"] != types.SpecStateFailed {
				t.Error("Eventually must honor the shared status timeout even with a SpecContext, before the spec times out")
			}
			var result Result
			readJSON(t, filepath.Join(directory, "result.json"), &result)
			if !shouldFail {
				passed, skipped := 5, 0
				if fixture == targetWithoutReference {
					passed, skipped = 3, 2
					if states["0022"] != types.SpecStateSkipped || states["0029"] != types.SpecStateSkipped {
						t.Fatalf("expected native skips for missing parameter reference: %v", states)
					}
				}
				if result.Passed != passed || result.Executed != passed || result.Skipped != skipped || result.Failed != 0 {
					t.Fatalf("unexpected result counts: %+v", result)
				}
			}
			checkTargetReportArtifacts(t, directory, shouldFail)
			if strings.Contains(string(output), "sensitive-target-value") {
				t.Fatal("console output exposed provider credentials or an API error")
			}
		})
	}
}

func checkTargetReportArtifacts(t *testing.T, directory string, shouldFail bool) {
	t.Helper()
	junitData, err := os.ReadFile(filepath.Join(directory, "junit.xml"))
	if err != nil {
		t.Fatal(err)
	}
	var junit reporters.JUnitTestSuites
	if err := xml.Unmarshal(junitData, &junit); err != nil || len(junit.TestSuites) != 1 {
		t.Fatalf("invalid JUnit report: %v", err)
	}
	if (junit.Failures+junit.Errors > 0) != shouldFail {
		t.Fatalf("JUnit status disagrees with process outcome: %+v", junit)
	}
	for _, name := range []string{"report.json", "junit.xml"} {
		data, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil || strings.Contains(string(data), "sensitive-target-value") {
			t.Fatalf("missing or unsafe report %s: %v", name, err)
		}
	}
}
