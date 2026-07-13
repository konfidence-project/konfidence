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
	"github.com/konfidence-project/konfidence/internal/vectorpromotion/internal/promotion"
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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	failClientCreationSecret = "fail-client-creation"

	vectorSigName            = "v-sig-A"
	ociCredSecretName        = "oci-credentials"
	sourceOnlyCredSecretName = "oci-credentials-source-only"
	signingCredSecretName    = "ocm-signing-creds"
)

var (
	ctx    context.Context
	cancel context.CancelFunc

	testEnv   *envtest.Environment
	cfg       *rest.Config
	k8sClient client.Client

	sourceRegistryEndpoint string
	targetRegistryEndpoint string

	// ocmClient is the test-side OCI client for seeding Zot and post-reconcile assertions.
	ocmClient pkgocm.Client

	vectorSigningKey pki.RSAKeyPair

	credSecretNames = []string{signingCredSecretName, ociCredSecretName}
)

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

	zotImage := fmt.Sprintf("ghcr.io/project-zot/zot-linux-%s:latest", runtime.GOARCH)

	By("starting source OCI registry container")
	sourceZotConfigDir, err := filepath.Abs(filepath.Join(".", "test", "zot-config", "source"))
	Expect(err).NotTo(HaveOccurred())

	sourceContainer, err := testcontainers.Run(
		ctx, zotImage,
		testcontainers.WithExposedPorts("5100/tcp"),
		testcontainers.WithFiles(
			testcontainers.ContainerFile{HostFilePath: filepath.Join(sourceZotConfigDir, "zot-config.json"), ContainerFilePath: "/etc/zot/config.json", FileMode: 0o644},
			testcontainers.ContainerFile{HostFilePath: filepath.Join(sourceZotConfigDir, "htpasswd"), ContainerFilePath: "/etc/zot/htpasswd", FileMode: 0o644},
		),
		testcontainers.WithWaitStrategy(
			wait.ForHTTP("/v2/").WithPort("5100/tcp").WithStatusCodeMatcher(func(s int) bool {
				return s == http.StatusOK || s == http.StatusUnauthorized
			}),
		),
	)
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func() { Expect(testcontainers.TerminateContainer(sourceContainer)).To(Succeed()) })

	sourceRegistryEndpoint, err = sourceContainer.Endpoint(ctx, "")
	Expect(err).NotTo(HaveOccurred())
	GinkgoWriter.Printf("Source OCI registry running at: http://%s\n", sourceRegistryEndpoint)

	By("starting target OCI registry container")
	targetZotConfigDir, err := filepath.Abs(filepath.Join(".", "test", "zot-config", "target"))
	Expect(err).NotTo(HaveOccurred())

	targetContainer, err := testcontainers.Run(
		ctx, zotImage,
		testcontainers.WithExposedPorts("5200/tcp"),
		testcontainers.WithFiles(
			testcontainers.ContainerFile{HostFilePath: filepath.Join(targetZotConfigDir, "zot-config.json"), ContainerFilePath: "/etc/zot/config.json", FileMode: 0o644},
			testcontainers.ContainerFile{HostFilePath: filepath.Join(targetZotConfigDir, "htpasswd"), ContainerFilePath: "/etc/zot/htpasswd", FileMode: 0o644},
		),
		testcontainers.WithWaitStrategy(
			wait.ForHTTP("/v2/").WithPort("5200/tcp").WithStatusCodeMatcher(func(s int) bool {
				return s == http.StatusOK || s == http.StatusUnauthorized
			}),
		),
	)
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func() { Expect(testcontainers.TerminateContainer(targetContainer)).To(Succeed()) })

	targetRegistryEndpoint, err = targetContainer.Endpoint(ctx, "")
	Expect(err).NotTo(HaveOccurred())
	GinkgoWriter.Printf("Target OCI registry running at: http://%s\n", targetRegistryEndpoint)

	By("bootstrapping envtest")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "..", "test", "data", "crds", "galaxy")},
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

	By("generating PKI key pair")
	vectorSigningKey = pki.GenerateRSAKeyPair("vector-signing-key")

	By("creating credential Secrets")
	tempClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())

	Expect(tempClient.Create(ctx, testocm.OCMConfigSecret(signingCredSecretName, "default",
		testocm.Bind(vectorSigName, vectorSigningKey),
	))).To(Succeed())
	Expect(tempClient.Create(ctx, testocm.DockerConfigSecret(ociCredSecretName, "default", "user", "password",
		sourceRegistryEndpoint, targetRegistryEndpoint,
	))).To(Succeed())
	Expect(tempClient.Create(ctx, testocm.DockerConfigSecret(sourceOnlyCredSecretName, "default", "user", "password",
		sourceRegistryEndpoint,
	))).To(Succeed())

	// The failClientCreationSecret is a magic-name Secret that causes the wrapped factory
	// to return an error. It must exist so the k8s client can reference it.
	Expect(tempClient.Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: failClientCreationSecret, Namespace: "default"},
	})).To(Succeed())

	By("building test-side OCM client for seeding and assertions")
	resolver, err := credentials.ResolverFromRefs(ctx, tempClient, "default", []credentials.Ref{{Name: ociCredSecretName}})
	Expect(err).NotTo(HaveOccurred())
	ocmClient, err = pkgocm.NewOciClientBuilder().WithLogger(ctrl.Log.WithName("ocm-client")).WithResolver(resolver).Build(ctx)
	Expect(err).NotTo(HaveOccurred())

	By("starting manager")
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
	log := ctrl.Log.WithName("vectorpromotion")

	// Wrap the production factory with the failClientCreationSecret guard used by
	// existing tests to simulate credential resolution failures.
	baseFactory := NewCacheFactory(log, limiter)
	wrappedFactory := clientcache.Factory[*konfidence.VectorPromotionConfig, promotion.OcmPort](
		func(ctx context.Context, k8sReader client.Reader, cr *konfidence.VectorPromotionConfig) (promotion.OcmPort, error) {
			if cr.Spec.Credentials != nil && cr.Spec.Credentials.OCM != nil {
				for _, ref := range cr.Spec.Credentials.OCM.Refs {
					if ref.Name == failClientCreationSecret {
						return nil, fmt.Errorf("failed to create OCM client: simulated credential resolution failure")
					}
				}
			}
			return baseFactory(ctx, k8sReader, cr)
		},
	)

	promotionCache, err := clientcache.New(
		clientcache.DefaultClientCacheSize,
		clientcache.DefaultExtract[*konfidence.VectorPromotionConfig],
		wrappedFactory,
	)
	Expect(err).NotTo(HaveOccurred())

	Expect(NewVectorPromotionReconciler(mgr, promotionCache).
		SetupWithManager(mgr)).To(Succeed())

	Expect(NewVectorPromotionTTLReconciler(mgr).
		SetupWithManager(mgr)).To(Succeed())

	Expect(NewVectorPromotionStatusPropagationReconciler(mgr).
		SetupWithManager(mgr)).To(Succeed())

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
