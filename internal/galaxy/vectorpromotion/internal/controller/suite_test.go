package controller_test

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

	galaxy "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/galaxy/vectorpromotion/internal/controller"
	pkgocm "github.com/konfidence-project/konfidence/pkg/ocm/repository"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
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

	"github.com/konfidence-project/konfidence/internal/galaxy/vectorpromotion/internal/ocm"
)

var (
	ctx    context.Context
	cancel context.CancelFunc

	testEnv   *envtest.Environment
	cfg       *rest.Config
	k8sClient client.Client

	sourceRegistryEndpoint string
	targetRegistryEndpoint string

	ocmClient pkgocm.Client
)

const failClientCreationSecret = "fail-client-creation"

func TestIntegrationController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Test Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx, cancel = context.WithCancel(context.TODO())
	DeferCleanup(func() {
		By("stopping manager")
		cancel()
	})

	By("starting source OCI registry container")
	sourceZotConfigDir, err := filepath.Abs(filepath.Join(".", "test", "zot-config", "source"))
	Expect(err).NotTo(HaveOccurred(), "failed to resolve source zot-config directory path")

	zotImage := fmt.Sprintf("ghcr.io/project-zot/zot-linux-%s:latest", runtime.GOARCH)

	sourceContainer, err := testcontainers.Run(
		ctx,
		zotImage,
		testcontainers.WithName("zot-promotion-source"),
		testcontainers.WithExposedPorts("5100/tcp"),
		testcontainers.WithFiles(
			testcontainers.ContainerFile{
				HostFilePath:      filepath.Join(sourceZotConfigDir, "zot-config.json"),
				ContainerFilePath: "/etc/zot/config.json",
				FileMode:          0o644,
			},
			testcontainers.ContainerFile{
				HostFilePath:      filepath.Join(sourceZotConfigDir, "htpasswd"),
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
	Expect(err).NotTo(HaveOccurred(), "failed to start source OCI registry container")
	DeferCleanup(func() {
		By("terminating source OCI registry container")
		Expect(testcontainers.TerminateContainer(sourceContainer)).To(Succeed())
	})

	sourceRegistryEndpoint, err = sourceContainer.Endpoint(ctx, "")
	Expect(err).NotTo(HaveOccurred(), "failed to get source OCI registry container endpoint")
	GinkgoWriter.Printf("Source OCI registry running at: http://%s\n", sourceRegistryEndpoint)

	By("starting target OCI registry container")
	targetZotConfigDir, err := filepath.Abs(filepath.Join(".", "test", "zot-config", "target"))
	Expect(err).NotTo(HaveOccurred(), "failed to resolve target zot-config directory path")

	targetContainer, err := testcontainers.Run(
		ctx,
		zotImage,
		testcontainers.WithName("zot-promotion-target"),
		testcontainers.WithExposedPorts("5200/tcp"),
		testcontainers.WithFiles(
			testcontainers.ContainerFile{
				HostFilePath:      filepath.Join(targetZotConfigDir, "zot-config.json"),
				ContainerFilePath: "/etc/zot/config.json",
				FileMode:          0o644,
			},
			testcontainers.ContainerFile{
				HostFilePath:      filepath.Join(targetZotConfigDir, "htpasswd"),
				ContainerFilePath: "/etc/zot/htpasswd",
				FileMode:          0o644,
			},
		),
		testcontainers.WithWaitStrategy(
			wait.ForHTTP("/v2/").
				WithPort("5200/tcp").
				WithStatusCodeMatcher(func(status int) bool {
					return status == http.StatusOK || status == http.StatusUnauthorized
				}),
		),
	)
	Expect(err).NotTo(HaveOccurred(), "failed to start target OCI registry container")
	DeferCleanup(func() {
		By("terminating target OCI registry container")
		Expect(testcontainers.TerminateContainer(targetContainer)).To(Succeed())
	})

	targetRegistryEndpoint, err = targetContainer.Endpoint(ctx, "")
	Expect(err).NotTo(HaveOccurred(), "failed to get target OCI registry container endpoint")
	GinkgoWriter.Printf("Target OCI registry running at: http://%s\n", targetRegistryEndpoint)

	By("bootstrapping envtest")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "..", "..", "api", "galaxy", "config", "bases", "crd")},
		ErrorIfCRDPathMissing: true,
		UseExistingCluster:    new(false),
	}
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
	ocmConfig := buildOCMConfig(
		registryCredential{endpoint: sourceRegistryEndpoint, username: "user", password: "password"},
		registryCredential{endpoint: targetRegistryEndpoint, username: "user", password: "password"},
	)
	ocmClient, err = pkgocm.NewOciClientBuilder().
		WithLogger(ctrl.Log.WithName("ocm-client")).
		WithOCMConfig(ocmConfig).
		Build(ctx)
	Expect(err).NotTo(HaveOccurred(), "failed to build OCM client")

	By("starting manager")
	startManager()
})

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

	Expect((&controller.VectorPromotionReconciler{
		Mgr:    mgr,
		Scheme: mgr.GetLocalManager().GetScheme(),
		OcmClientProvider: pkgocm.ClientProviderFunc(
			func(_ context.Context, _ client.Reader, _ string, creds []galaxy.CredentialsConfig) (pkgocm.Client, error) {
				for _, c := range creds {
					if c.Name == failClientCreationSecret {
						return nil, fmt.Errorf("simulated credential resolution failure")
					}
				}
				return ocmClient, nil
			},
		),
		PortProvider: ocm.NewPromotionPortProvider(),
	}).SetupWithManager(mgr)).To(Succeed())

	Expect((&controller.VectorPromotionTTLReconciler{
		Mgr:    mgr,
		Scheme: mgr.GetLocalManager().GetScheme(),
	}).SetupWithManager(mgr)).To(Succeed())

	Expect((&controller.VectorPromotionStatusPropagationReconciler{
		Mgr:    mgr,
		Scheme: mgr.GetLocalManager().GetScheme(),
	}).SetupWithManager(mgr)).To(Succeed())

	managerCtx, managerCancel := context.WithCancel(ctx)
	go func() {
		defer GinkgoRecover()
		Expect(mgr.Start(managerCtx)).To(Succeed())
	}()

	DeferCleanup(managerCancel)

	Eventually(func() bool {
		return mgr.GetLocalManager().GetCache().WaitForCacheSync(ctx)
	}).Should(BeTrue())
}

type registryCredential struct {
	endpoint string
	username string
	password string
}

func buildOCMConfig(endpoints ...registryCredential) *configuration.Configuration {
	auths := make(map[string]any, len(endpoints))
	for _, ep := range endpoints {
		auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", ep.username, ep.password)))
		auths[ep.endpoint] = map[string]any{"auth": auth}
	}
	dockerConfigJSON, err := json.Marshal(map[string]any{"auths": auths})
	Expect(err).NotTo(HaveOccurred(), "failed to marshal docker config JSON")

	repoRaw := &ocmruntime.Raw{
		Data: fmt.Appendf(nil, `{
			"type": "DockerConfig/v1",
			"dockerConfig": %q
		}`, string(dockerConfigJSON)),
	}

	credConfig := &credentialsv1.Config{
		Repositories: []credentialsv1.RepositoryConfigEntry{{Repository: repoRaw}},
	}

	credScheme := ocmruntime.NewScheme()
	credentialsv1.MustRegister(credScheme)

	rawCreds := &ocmruntime.Raw{}
	err = credScheme.Convert(credConfig, rawCreds)
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
