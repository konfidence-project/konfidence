package suite

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/konfidence-project/konfidence/internal/conformance/setup"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func testScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	return scheme
}

func TestRegistrySecretLifecycle(t *testing.T) {
	const dockerConfig = `{"auths":{"registry.example.com":{"auth":"dXNlcjpwYXNz"}}}`
	ocmConfig := fmt.Sprintf(`{
		"type":"generic.config.ocm.software/v1",
		"configurations":[{
			"type":"credentials.config.ocm.software/v1",
			"repositories":[{"repository":{"type":"DockerConfig/v1","dockerConfig":%q}}]
		}]
	}`, dockerConfig)
	for key, data := range map[string]string{".dockerconfigjson": dockerConfig, ".ocmconfig": ocmConfig} {
		t.Run(key, func(t *testing.T) {
			source := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "registry"},
				StringData: map[string]string{key: data},
			}
			inputs := &setup.Inputs{
				RegistrySecrets: []*corev1.Secret{source},
			}
			foreign := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "registry", Namespace: "developer-owned"}}
			var namespace string
			var fetched, cleaned bool
			k8sClient := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(foreign).
				WithInterceptorFuncs(interceptor.Funcs{
					Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
						if secret, ok := obj.(*corev1.Secret); ok {
							// Simulate the API server's stringData -> data conversion.
							if secret.Namespace == "" {
								t.Fatal("registry Secret has no shared namespace")
							}
							secret.Data = make(map[string][]byte)
							for k, v := range secret.StringData {
								secret.Data[k] = []byte(v)
							}
							secret.StringData = nil
							namespace = secret.Namespace
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
						cleaned = obj.GetName() == namespace
						return c.Delete(ctx, obj, opts...)
					},
				}).Build()
			result, err := runWithClient(context.Background(), inputs, k8sClient)
			if err != nil || result.Executed != 0 {
				t.Fatalf("unexpected suite result: %+v, %v", result, err)
			}
			if !fetched || !cleaned {
				t.Fatalf("expected shared resolver to fetch created Secret and cleanup namespace: fetched=%t cleaned=%t", fetched, cleaned)
			}
			if source.Namespace != "" || source.StringData[key] != data || source.Data != nil {
				t.Fatal("suite mutated caller-owned registry Secret")
			}
			namespaces := &corev1.NamespaceList{}
			if err := k8sClient.List(context.Background(), namespaces); err != nil || len(namespaces.Items) != 0 {
				t.Fatalf("shared namespace was not deleted: %v", err)
			}
			if err := k8sClient.Get(context.Background(), client.ObjectKeyFromObject(foreign), &corev1.Secret{}); err != nil {
				t.Fatalf("developer-owned Secret was affected: %v", err)
			}
		})
	}
}

func TestRegistrySetupFailuresCleanUp(t *testing.T) {
	for _, failure := range []string{"create-secret", "decode-credentials", "cancellation", "cleanup"} {
		t.Run(failure, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			source := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "registry"},
				Data: map[string][]byte{
					".dockerconfigjson": []byte(`{"auths":{}}`),
				},
			}
			if failure == "decode-credentials" {
				source.Data[".dockerconfigjson"] = []byte("sensitive-password")
			}
			cleaned := false
			k8sClient := fake.NewClientBuilder().WithScheme(testScheme(t)).
				WithInterceptorFuncs(interceptor.Funcs{
					Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
						if _, ok := obj.(*corev1.Secret); ok {
							if failure == "create-secret" {
								return errors.New("API rejected sensitive-password")
							}
							if failure == "cancellation" {
								cancel()
								return ctx.Err()
							}
						}
						return c.Create(ctx, obj, opts...)
					},
					Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
						cleaned = true
						if ctx.Err() != nil {
							t.Fatal("cleanup used canceled context")
						}
						if failure == "cleanup" {
							return errors.New("sensitive-cleanup-error")
						}
						return c.Delete(ctx, obj, opts...)
					},
				}).Build()
			_, err := runWithClient(ctx, &setup.Inputs{RegistrySecrets: []*corev1.Secret{source}}, k8sClient)
			if err == nil || !cleaned {
				t.Fatalf("expected failure and cleanup: error=%v cleaned=%t", err, cleaned)
			}
			if strings.Contains(err.Error(), "sensitive") {
				t.Fatal("error exposed credential contents")
			}
		})
	}
}
