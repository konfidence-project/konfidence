package suite

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/konfidence-project/konfidence/internal/conformance/setup"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const (
	registryCleanupFailure = "cleanup"
	registryEmptySuite     = "empty"
)

// TestRegistryFixture exercises registry resources through real Ginkgo lifecycle
// hooks. Each parent test launches a separate suite process.
func TestRegistryFixture(t *testing.T) {
	fixture := os.Getenv("KDEN_CONFORMANCE_REGISTRY_FIXTURE")
	if fixture == "" {
		t.Skip("subprocess fixture")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	const dockerConfig = `{"auths":{"registry.example.com":{"auth":"dXNlcjpwYXNz"}}}`
	key, data := ".dockerconfigjson", dockerConfig
	if fixture == "ocmconfig" {
		key = ".ocmconfig"
		data = fmt.Sprintf(`{
			"type":"generic.config.ocm.software/v1",
			"configurations":[{
				"type":"credentials.config.ocm.software/v1",
				"repositories":[{"repository":{"type":"DockerConfig/v1","dockerConfig":%q}}]
			}]
		}`, dockerConfig)
	}
	if fixture == "decode-credentials" {
		data = "sensitive-password"
	}
	source := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "registry"},
		StringData: map[string]string{key: data},
	}
	foreign := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "registry", Namespace: "developer-owned"}}
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	var namespace string
	var fetched, cleaned, created bool
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(foreign).
		WithInterceptorFuncs(interceptor.Funcs{
			Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				if _, ok := obj.(*corev1.Namespace); ok {
					created = true
					if fixture == "create-namespace" {
						return errors.New("sensitive-namespace-response")
					}
				}
				if secret, ok := obj.(*corev1.Secret); ok {
					namespace = secret.Namespace
					if fixture == "create-secret" {
						return errors.New("sensitive-secret-response")
					}
					if fixture == "cancellation" {
						cancel()
						return context.Canceled
					}
					// Simulate the API server's stringData conversion.
					secret.Data = make(map[string][]byte)
					for k, v := range secret.StringData {
						secret.Data[k] = []byte(v)
					}
					secret.StringData = nil
				}
				return c.Create(ctx, obj, opts...)
			},
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if _, ok := obj.(*corev1.Secret); ok {
					fetched = key.Name == source.Name && key.Namespace == namespace
				}
				return c.Get(ctx, key, obj, opts...)
			},
			Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
				if ctx.Err() != nil {
					t.Error("Ginkgo cleanup received a canceled context")
				}
				cleaned = true
				if fixture == registryCleanupFailure {
					return errors.New("sensitive-cleanup-response")
				}
				return c.Delete(ctx, obj, opts...)
			},
		}).Build()
	environment := &Environment{
		Context: ctx, Client: k8sClient,
		Inputs: &setup.Inputs{RegistrySecrets: []*corev1.Secret{source}},
	}
	result, runErr := runFramework(environment, Options{ReportDir: os.Getenv("KDEN_CONFORMANCE_FIXTURE_REPORT_DIR")},
		func(env *Environment) {
			registerLifecycle(env)
			if fixture == registryEmptySuite {
				return
			}
			ginkgo.It("uses shared registry credentials", func() {
				gomega.Expect(env.SharedNamespace).NotTo(gomega.BeNil())
				gomega.Expect(env.RegistryCredentials).NotTo(gomega.BeNil())
				gomega.Expect(fetched).To(gomega.BeTrue())
				gomega.Expect(source.Namespace).To(gomega.BeEmpty())
				gomega.Expect(source.StringData[key]).To(gomega.Equal(data))
				gomega.Expect(source.Data).To(gomega.BeNil())
			})
		})
	if fixture == registryEmptySuite {
		if created || cleaned {
			t.Fatal("an empty suite must not run resource lifecycle hooks")
		}
	} else if fixture != "create-namespace" && !cleaned {
		t.Fatal("Ginkgo did not clean up the shared namespace")
	}
	if fixture != registryCleanupFailure {
		namespaces := &corev1.NamespaceList{}
		if err := k8sClient.List(context.Background(), namespaces); err != nil || len(namespaces.Items) != 0 {
			t.Fatalf("shared namespace was not deleted: %v", err)
		}
	}
	if err := k8sClient.Get(context.Background(), client.ObjectKeyFromObject(foreign), &corev1.Secret{}); err != nil {
		t.Fatalf("developer-owned Secret was affected: %v", err)
	}
	if result.ReportDir == "" {
		t.Fatal("expected the native report directory")
	}
	if runErr != nil {
		t.Fatal(runErr)
	}
}

func TestRegistryLifecycleReports(t *testing.T) {
	for _, fixture := range []string{
		"dockerconfig", "ocmconfig", "create-namespace", "create-secret",
		"decode-credentials", "cancellation", registryCleanupFailure, registryEmptySuite,
	} {
		t.Run(fixture, func(t *testing.T) {
			directory := t.TempDir()
			command := exec.Command(os.Args[0], "-test.run=^TestRegistryFixture$")
			command.Env = append(os.Environ(), "KDEN_CONFORMANCE_REGISTRY_FIXTURE="+fixture,
				"KDEN_CONFORMANCE_FIXTURE_REPORT_DIR="+directory)
			output, err := command.CombinedOutput()
			shouldFail := fixture != "dockerconfig" && fixture != "ocmconfig" && fixture != registryEmptySuite
			if (err != nil) != shouldFail {
				t.Fatalf("unexpected subprocess outcome: %v\n%s", err, output)
			}
			var reports []Report
			readJSON(t, filepath.Join(directory, "report.json"), &reports)
			if len(reports) != 1 || reports[0].SuiteSucceeded == shouldFail {
				t.Fatalf("unexpected framework status: %+v", reports)
			}
			if shouldFail {
				node := types.NodeTypeBeforeSuite
				if fixture == registryCleanupFailure {
					node = types.NodeTypeCleanupAfterSuite
				}
				if reports[0].SpecReports.WithLeafNodeType(node).CountWithState(types.SpecStateFailureStates) != 1 {
					t.Fatalf("expected a failed native %s node", node)
				}
			}
			for _, name := range []string{"report.json", "junit.xml"} {
				data, err := os.ReadFile(filepath.Join(directory, name))
				if err != nil || strings.Contains(string(data), "sensitive-") {
					t.Fatalf("missing or unsafe report %s: %v", name, err)
				}
			}
		})
	}
}
