package controller

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	galaxy "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/galaxy/vectorassembly/internal/ocm"
	"github.com/konfidence-project/konfidence/internal/galaxy/vectorassembly/internal/vector"
	pkgocm "github.com/konfidence-project/konfidence/pkg/ocm/repository"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"k8s.io/client-go/kubernetes/scheme"
	genericv1 "ocm.software/open-component-model/bindings/go/configuration/generic/v1/spec"
	credentialsv1 "ocm.software/open-component-model/bindings/go/credentials/spec/config/v1"
	ocmruntime "ocm.software/open-component-model/bindings/go/runtime"
	"ocm.software/open-component-model/kubernetes/controller/pkg/configuration"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"

	"k8s.io/client-go/rest"
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
	ocmClient pkgocm.Client
	// testVersion is the version returned by the static version generator used in tests for newly created vectors
	testVersion = "2026.1.2-000000000Z"
	// oldTestVersion is used when manually pushing pre-existing vectors in test setup (to simulate existing state)
	oldTestVersion = "2026.1.1-000000000Z"
	// testVersionGenerator is a static version generator that always returns testVersion, used in tests to have predictable vector versions
	testVersionGenerator = vector.VersionGeneratorFunc(func() string {
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
	zotConfigDir, err := filepath.Abs(filepath.Join(".", "test", "zot-config"))
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
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "..", "..", "api", "galaxy", "config", "bases", "crd")},
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
	Expect(galaxy.AddToScheme(scheme.Scheme)).To(Succeed())

	By("building shared OCM client")
	ocmConfig := buildOCMConfig(registryEndpoint, "user", "password")
	ocmClient, err = pkgocm.NewOciClientBuilder().
		WithLogger(ctrl.Log.WithName("ocm-client")).
		WithOCMConfig(ocmConfig).
		Build(ctx)
	Expect(err).NotTo(HaveOccurred(), "failed to build OCM client")

	By("starting manager with OCM adapter")
	startManager()

})

// startManager wires up the VectorTemplateReconciler with providers,
// then starts the manager in a background goroutine.
func startManager() {
	var err error

	mgr, err := mcmanager.New(cfg, nil, ctrl.Options{
		Scheme: scheme.Scheme,
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
	})
	Expect(err).NotTo(HaveOccurred(), "failed to create multicluster manager")

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred(), "failed to create k8s client")

	Expect((&VectorTemplateReconciler{
		Mgr:    mgr,
		Scheme: mgr.GetLocalManager().GetScheme(),
		OcmClientProvider: pkgocm.ClientProviderFunc(
			func(
				_ context.Context,
				_ client.Reader,
				_ string,
				_ []galaxy.CredentialsConfig,
			) (pkgocm.Client, error) {
				return ocmClient, nil
			},
		),
		VectorOcmPortProvider: ocm.NewPortProvider(),
		VersionGenerator:      testVersionGenerator,
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

// buildOCMConfig creates an OCM Configuration with DockerConfig credentials for the given registry endpoint.
func buildOCMConfig(endpoint, username, password string) *configuration.Configuration {
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password)))
	dockerConfigJSON := fmt.Sprintf(`{"auths":{%q:{"auth":%q}}}`, endpoint, auth)

	repoRaw := &ocmruntime.Raw{
		Data: []byte(fmt.Sprintf(`{
			"type": "DockerConfig/v1",
			"dockerConfig": %q
		}`, dockerConfigJSON)),
	}

	credConfig := &credentialsv1.Config{
		Repositories: []credentialsv1.RepositoryConfigEntry{{Repository: repoRaw}},
	}

	credScheme := ocmruntime.NewScheme()
	credentialsv1.MustRegister(credScheme)

	rawCreds := &ocmruntime.Raw{}
	err := credScheme.Convert(credConfig, rawCreds)
	Expect(err).NotTo(HaveOccurred(), "failed to convert credentials config")

	cfg := &genericv1.Config{
		Type: ocmruntime.Type{
			Version: genericv1.Version,
			Name:    genericv1.ConfigType,
		},
		Configurations: []*ocmruntime.Raw{rawCreds},
	}

	return &configuration.Configuration{Config: cfg}
}

// envtest needs the etcd and kube-apiserver binaries that `make setup-envtest` downloads
// into bin/k8s/<version>-<os>-<arch>/. When running via the Makefile, KUBEBUILDER_ASSETS
// is set and envtest finds them automatically. When running from an IDE, that env var is
// absent. This helper bridges the gap by scanning the known download location so IDE-based
// test runs work without extra configuration.
func getFirstFoundEnvTestBinaryDir() string {
	basePath := filepath.Join("..", "..", "..", "..", "..", "bin", "k8s")
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
