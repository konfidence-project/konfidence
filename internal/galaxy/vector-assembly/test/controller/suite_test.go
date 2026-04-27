/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	global "github.com/konfidence-project/crds/api/global/v1alpha1"
	pkgOcm "github.com/konfidence-project/pkg/ocm/repository"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"

	"k8s.io/client-go/rest"

	"github.com/konfidence-project/gcp-vector-assembly-controller/internal/controller"
	"github.com/konfidence-project/gcp-vector-assembly-controller/internal/controller/domain"
	"github.com/konfidence-project/gcp-vector-assembly-controller/pkg/ocm"
)

var (
	ctx       context.Context
	cancel    context.CancelFunc
	testEnv   *envtest.Environment
	cfg       *rest.Config
	k8sClient client.Client
	// registryEndpoint is the host:port of the oci container, e.g. "localhost:55123"
	registryEndpoint string
	// ocmClient is the shared OCM client used for both seeding test data and by the adapter
	ocmClient pkgOcm.Client
	// testVersion is the version returned by the static version generator used in tests for newly created vectors
	testVersion = "2026.1.2-000000000Z"
	// oldTestVersion is used when manually pushing pre-existing vectors in test setup (to simulate existing state)
	oldTestVersion = "2026.1.1-000000000Z"
	// testVersionGenerator is a static version generator that always returns testVersion, used in tests to have predictable vector versions
	testVersionGenerator = domain.VectorVersionGeneratorFunc(func() string {
		return testVersion
	})
)

func TestController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx, cancel = context.WithCancel(context.TODO())
	DeferCleanup(func() {
		By("stopping manager")
		cancel()
	})

	By("starting OCI registry container")
	zotConfigDir, err := filepath.Abs(filepath.Join(".", "zot-config"))
	Expect(err).NotTo(HaveOccurred(), "failed to resolve zot-config directory path")

	// Use the full Zot image (includes web UI) matching the host architecture.
	zotImage := fmt.Sprintf("ghcr.io/project-zot/zot-linux-%s:latest", runtime.GOARCH)

	ociContainer, err := testcontainers.Run(
		ctx,
		zotImage,
		testcontainers.WithName("zot-controller-test"),
		testcontainers.WithExposedPorts("5100/tcp"),
		testcontainers.WithFiles(
			testcontainers.ContainerFile{
				HostFilePath:      filepath.Join(zotConfigDir, "zot-config.json"),
				ContainerFilePath: "/etc/zot/config.json",
				FileMode:          0o644,
			},
			testcontainers.ContainerFile{
				HostFilePath:      filepath.Join(zotConfigDir, "htpasswd"),
				ContainerFilePath: "/etc/zot/htpasswd",
				FileMode:          0o644,
			},
		),
		testcontainers.WithWaitStrategy(
			wait.ForHTTP("/v2/").
				WithPort("5100/tcp").
				WithStatusCodeMatcher(func(status int) bool {
					return status == http.StatusOK || status == http.StatusUnauthorized
				}),
		),
	)
	Expect(err).NotTo(HaveOccurred(), "failed to start OCI registry container")
	DeferCleanup(func() {
		By("terminating OCI registry container")
		Expect(testcontainers.TerminateContainer(ociContainer)).To(Succeed())
	})

	registryEndpoint, err = ociContainer.Endpoint(ctx, "")
	Expect(err).NotTo(HaveOccurred(), "failed to get OCI registry container endpoint")
	GinkgoWriter.Printf("OCI registry running at: http://%s\n", registryEndpoint)

	By("bootstrapping envtest")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "test", "data", "generated", "crds")},
		ErrorIfCRDPathMissing: true,
		UseExistingCluster:    new(false),
	}
	// getFirstFoundEnvTestBinaryDir is an IDE convenience: when running tests from an IDE
	// (GoLand, VS Code), the KUBEBUILDER_ASSETS env var is not set because the Makefile
	// is not involved. This helper scans bin/k8s/ for binaries that `make setup-envtest`
	// downloaded earlier, so envtest can locate them without manual env configuration.
	// When KUBEBUILDER_ASSETS is already set (e.g. via Makefile), envtest uses that and
	// this helper is effectively a no-op.
	if dir := getFirstFoundEnvTestBinaryDir(); dir != "" {
		testEnv.BinaryAssetsDirectory = dir
	}

	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred(), "failed to start envtest")
	Expect(cfg).NotTo(BeNil(), "envtest config should not be nil")
	DeferCleanup(func() {
		By("tearing down envtest")
		Expect(testEnv.Stop()).To(Succeed())
	})

	By("registering schemes")
	Expect(global.AddToScheme(scheme.Scheme)).To(Succeed())

	By("building shared OCM client")
	registrySecret := buildRegistrySecret(registryEndpoint, "user", "password")
	ocmClient, err = pkgOcm.NewOciClientBuilder().
		WithLogger(ctrl.Log.WithName("ocm-client")).
		WithDockerConfigJsonSecret(registrySecret).
		Build(ctx)
	Expect(err).NotTo(HaveOccurred(), "failed to build OCM client")

	By("starting manager with OCM adapter")
	ocmAdapter := ocm.NewAdapter(
		ocm.WithOcmClient(ctx, registrySecret),
	)
	startManager(ocmAdapter)
})

// startManager wires up the VectorTemplateReconciler with the given OCM adapter,
// then starts the manager in a background goroutine.
func startManager(ocmAdapter domain.VectorOcmPort) {
	var err error

	mgr, err := mcmanager.New(cfg, nil, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).NotTo(HaveOccurred(), "failed to create multicluster manager")

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred(), "failed to create k8s client")

	Expect((&controller.VectorTemplateReconciler{
		Mgr:              mgr,
		Scheme:           mgr.GetLocalManager().GetScheme(),
		OcmAdapter:       ocmAdapter,
		VersionGenerator: testVersionGenerator,
	}).SetupWithManager(mgr)).To(Succeed())

	managerCtx, managerCancel := context.WithCancel(ctx)
	go func() {
		defer GinkgoRecover()
		Expect(mgr.Start(managerCtx)).To(Succeed())
	}()

	// Register manager cancel to run when the suite context is done
	DeferCleanup(managerCancel)

	Eventually(func() bool {
		return mgr.GetLocalManager().GetCache().WaitForCacheSync(ctx)
	}).Should(BeTrue())
}

// buildRegistrySecret creates a kubernetes.io/dockerconfigjson Secret for the given registry endpoint.
func buildRegistrySecret(endpoint, username, password string) *corev1.Secret {
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password)))
	dockerConfig := map[string]any{
		"auths": map[string]any{
			endpoint: map[string]any{
				"auth": auth,
			},
		},
	}
	dockerConfigJSON, err := json.Marshal(dockerConfig)
	Expect(err).NotTo(HaveOccurred(), "failed to marshal docker config JSON")
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "oci-registry-credentials"},
		Type:       corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: dockerConfigJSON,
		},
	}
}

// envtest needs the etcd and kube-apiserver binaries that `make setup-envtest` downloads
// into bin/k8s/<version>-<os>-<arch>/. When running via the Makefile, KUBEBUILDER_ASSETS
// is set and envtest finds them automatically. When running from an IDE, that env var is
// absent. This helper bridges the gap by scanning the known download location so IDE-based
// test runs work without extra configuration.
func getFirstFoundEnvTestBinaryDir() string {
	basePath := filepath.Join("..", "..", "bin", "k8s")
	entries, err := os.ReadDir(basePath)
	if err != nil {
		logf.Log.Error(err, "Failed to read envtest binary directory", "path", basePath)
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return filepath.Join(basePath, entry.Name())
		}
	}
	return ""
}
