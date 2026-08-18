package controller

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/vectorassembly/internal/vector"
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
	// client per reconcile from the credential Secret via buildAdapter.
	ocmClient pkgocm.Client

	// testVersionGenerator returns a unique, monotonically increasing concrete version on
	// every call, mirroring the production timestamp generator (which never repeats). A
	// fixed version would collide on re-assembly: the OCI Save skips an already-existing
	// name+version, so the descriptor would never be overwritten and drift could not be
	// observed. Tests capture the produced version via status.latestVector rather than
	// asserting a hardcoded constant.
	testVersionSeq       atomic.Int32
	testVersionGenerator = vector.VersionGeneratorFunc(func() string {
		return fmt.Sprintf("2026.1.%d-000000000Z", testVersionSeq.Add(1))
	})

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
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "..", "test", "data", "crds")},
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
	Expect(konfidence.AddToScheme(scheme.Scheme)).To(Succeed())

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
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:  scheme.Scheme,
		Metrics: metricsserver.Options{BindAddress: "0"},
	})
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())

	limiter := cryptopkg.NewLimiter(0)
	log := ctrl.Log.WithName("vectorassembly")

	// Build the shared verifier the same way cmd/konfidence/cmd/operator.go does
	// so envtest exercises the production wiring.
	sharedVerifier, err := cryptopkg.NewVerifierBuilder().
		WithParallelism(limiter).
		WithCache(1024, 30*time.Minute).
		WithLogger(log).
		Build()
	Expect(err).NotTo(HaveOccurred())

	vectorCache, err := lru.New[string, vector.Vector](VectorCacheSize)
	Expect(err).NotTo(HaveOccurred())

	reconciler := NewVectorTemplateReconciler(mgr, sharedVerifier, limiter, log, vectorCache, testVersionGenerator)
	reconciler.assemblyPollInterval = 100 * time.Millisecond
	Expect(reconciler.SetupWithManager(mgr)).To(Succeed())

	managerCtx, managerCancel := context.WithCancel(ctx)
	go func() {
		defer GinkgoRecover()
		Expect(mgr.Start(managerCtx)).To(Succeed())
	}()
	DeferCleanup(managerCancel)

	Eventually(func() bool {
		return mgr.GetCache().WaitForCacheSync(ctx)
	}).Should(BeTrue())
}

func getFirstFoundEnvTestBinaryDir() string {
	basePath := filepath.Join("..", "..", "..", "..", "bin", "k8s")
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
