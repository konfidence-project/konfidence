package controller

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	galaxy "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/galaxy/vectorassembly/internal/vector"
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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"

	"k8s.io/client-go/rest"
)

const (
	vectorSigName   = "v-sig-A"
	artifactSigName = "artifact-sig-B"

	ociCredSecretName     = "oci-credentials"
	signingCredSecretName = "ocm-signing-creds"
)

var (
	ctx       context.Context
	cancel    context.CancelFunc
	testEnv   *envtest.Environment
	cfg       *rest.Config
	k8sClient client.Client

	registryEndpoint string

	// ocmClient is the test-side OCI client used for seeding Zot with test data
	// and asserting post-reconcile descriptor state. The controller builds its own
	// client from the credential Secret via NewCacheFactory.
	ocmClient pkgocm.Client

	testVersion    = "2026.1.2-000000000Z"
	oldTestVersion = "2026.1.1-000000000Z"

	testVersionGenerator = vector.VersionGeneratorFunc(func() string { return testVersion })

	vectorSigningKey   pki.RSAKeyPair
	artifactSigningKey pki.RSAKeyPair

	// credSecretNames are the credential Secret names to wire into VectorTemplate CRs.
	credSecretNames = []string{signingCredSecretName, ociCredSecretName}
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

	By("bootstrapping envtest")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "..", "..", "test", "data", "crds", "galaxy")},
		ErrorIfCRDPathMissing: true,
		UseExistingCluster:    new(false),
	}
	if dir := getFirstFoundEnvTestBinaryDir(); dir != "" {
		testEnv.BinaryAssetsDirectory = dir
	}
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())
	DeferCleanup(func() { Expect(testEnv.Stop()).To(Succeed()) })

	By("registering schemes")
	Expect(galaxy.AddToScheme(scheme.Scheme)).To(Succeed())

	By("generating PKI key pairs")
	vectorSigningKey = pki.GenerateRSAKeyPair("vector-signing-key")
	artifactSigningKey = pki.GenerateRSAKeyPair("artifact-signing-key")

	By("creating credential Secrets in envtest")
	apiReader, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())

	Expect(apiReader.Create(ctx, testocm.OCMConfigSecret(signingCredSecretName, "default",
		testocm.Bind(vectorSigName, vectorSigningKey),
		testocm.Bind(artifactSigName, artifactSigningKey),
	))).To(Succeed())
	Expect(apiReader.Create(ctx, testocm.DockerConfigSecret(ociCredSecretName, "default", "user", "password", registryEndpoint))).To(Succeed())

	By("building test-side OCM client for seeding and assertions")
	resolver, err := credentials.ResolverFromRefs(ctx, apiReader, "default", []credentials.Ref{{Name: ociCredSecretName}})
	Expect(err).NotTo(HaveOccurred())
	ocmClient, err = pkgocm.NewOciClientBuilder().WithLogger(ctrl.Log.WithName("ocm-client")).WithResolver(resolver).Build(ctx)
	Expect(err).NotTo(HaveOccurred())

	By("starting manager with production PKI factory")
	startManager()
})

func startManager() {
	mgr, err := mcmanager.New(cfg, nil, ctrl.Options{
		Scheme:  scheme.Scheme,
		Metrics: metricsserver.Options{BindAddress: "0"},
	})
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())

	limiter := cryptopkg.NewLimiter(0)
	log := ctrl.Log.WithName("vectorassembly")

	cache, err := clientcache.New(
		clientcache.DefaultClientCacheSize,
		clientcache.DefaultExtract[*galaxy.VectorTemplate],
		NewCacheFactory(log, limiter),
	)
	Expect(err).NotTo(HaveOccurred())

	Expect((&VectorTemplateReconciler{
		Mgr:              mgr,
		Scheme:           mgr.GetLocalManager().GetScheme(),
		Cache:            cache,
		VersionGenerator: testVersionGenerator,
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
