package controller

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/pkg/ocm/clientcache"
	"github.com/konfidence-project/konfidence/pkg/ocm/credentials"
	cryptopkg "github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	pkgocm "github.com/konfidence-project/konfidence/pkg/ocm/repository"
	testocm "github.com/konfidence-project/konfidence/pkg/testutil/ocm"
	"github.com/konfidence-project/konfidence/pkg/testutil/pki"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

const (
	vectorSigName         = "v-sig-A"
	ociCredSecretName     = "oci-credentials"
	signingCredSecretName = "ocm-signing-creds"
)

var (
	ctx        context.Context
	cancel     context.CancelFunc
	testEnv    *envtest.Environment
	cfg        *rest.Config
	k8sClient  client.Client
	k8sManager ctrl.Manager

	registryEndpoint string

	// ocmClient is the test-side OCI client for seeding Zot and post-reconcile assertions.
	ocmClient pkgocm.Client

	vectorSigningKey pki.RSAKeyPair

	credSecretNames = []string{signingCredSecretName, ociCredSecretName}
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx, cancel = context.WithCancel(context.TODO())

	var err error
	Expect(konfidence.AddToScheme(scheme.Scheme)).To(Succeed())

	By("starting OCI registry container")
	zotConfigDir, err := filepath.Abs(filepath.Join(".", "test", "zot-config"))
	Expect(err).NotTo(HaveOccurred())

	zotImage := fmt.Sprintf("ghcr.io/project-zot/zot-linux-%s:latest", runtime.GOARCH)
	ociContainer, err := testcontainers.Run(
		ctx, zotImage,
		testcontainers.WithExposedPorts("5100/tcp"),
		testcontainers.WithFiles(
			testcontainers.ContainerFile{HostFilePath: filepath.Join(zotConfigDir, "zot-config.json"), ContainerFilePath: "/etc/zot/config.json", FileMode: 0o644},
			testcontainers.ContainerFile{HostFilePath: filepath.Join(zotConfigDir, "htpasswd"), ContainerFilePath: "/etc/zot/htpasswd", FileMode: 0o644},
		),
		testcontainers.WithWaitStrategy(
			wait.ForHTTP("/v2/").WithPort("5100/tcp").WithStatusCodeMatcher(func(s int) bool {
				return s == http.StatusOK || s == http.StatusUnauthorized
			}),
		),
	)
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func() { Expect(testcontainers.TerminateContainer(ociContainer)).To(Succeed()) })

	registryEndpoint, err = ociContainer.Endpoint(ctx, "")
	Expect(err).NotTo(HaveOccurred())
	GinkgoWriter.Printf("OCI registry running at: http://%s\n", registryEndpoint)

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "..", "..", "test", "data", "crds"),
		},
		ErrorIfCRDPathMissing: true,
	}
	if dir := getFirstFoundEnvTestBinaryDir(); dir != "" {
		testEnv.BinaryAssetsDirectory = dir
	}
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	By("generating PKI key pair")
	vectorSigningKey = pki.GenerateRSAKeyPair("vector-signing-key")

	By("creating credential Secrets")
	Expect(k8sClient.Create(ctx, testocm.OCMConfigSecret(signingCredSecretName, "default",
		testocm.Bind(vectorSigName, vectorSigningKey),
	))).To(Succeed())
	Expect(k8sClient.Create(ctx, testocm.DockerConfigSecret(ociCredSecretName, "default", "user", "password", registryEndpoint))).To(Succeed())

	By("building test-side OCM client for seeding and assertions")
	resolver, err := credentials.ResolverFromRefs(ctx, k8sClient, "default", []credentials.Ref{{Name: ociCredSecretName}})
	Expect(err).NotTo(HaveOccurred())
	ocmClient, err = pkgocm.NewOciClientBuilder().WithLogger(ctrl.Log.WithName("ocm-client")).WithResolver(resolver).Build(ctx)
	Expect(err).NotTo(HaveOccurred())

	By("creating target namespace for the Stage")
	createNamespace(ctx, k8sClient, "target")

	By("creating manager and wiring StageConfiguration controller")
	k8sManager, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme:  scheme.Scheme,
		Metrics: metricsserver.Options{BindAddress: "0"},
	})
	Expect(err).ToNot(HaveOccurred())

	limiter := cryptopkg.NewLimiter(0)
	log := ctrl.Log.WithName("stageconfiguration")

	cache, err := clientcache.New(
		clientcache.DefaultClientCacheSize,
		clientcache.DefaultExtract[*konfidence.StageConfiguration],
		NewCacheFactory(log, limiter),
	)
	Expect(err).NotTo(HaveOccurred())

	Expect(NewStageConfigurationReconciler(k8sManager, cache).SetupWithManager(k8sManager)).To(Succeed())

	go func() {
		defer GinkgoRecover()
		Expect(k8sManager.Start(ctx)).ToNot(HaveOccurred(), "failed to run manager")
	}()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	Expect(testEnv.Stop()).To(Succeed())
})

func getFirstFoundEnvTestBinaryDir() string {
	basePath := filepath.Join("..", "..", "..", "..", "bin", "k8s")
	entries, err := os.ReadDir(basePath)
	if err != nil {
		logf.Log.Error(err, "Failed to read directory", "path", basePath)
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return filepath.Join(basePath, entry.Name())
		}
	}
	return ""
}
