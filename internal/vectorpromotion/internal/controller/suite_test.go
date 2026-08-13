package controller

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var (
	ctx    context.Context
	cancel context.CancelFunc

	testEnv   *envtest.Environment
	cfg       *rest.Config
	k8sClient client.Client
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

	By("bootstrapping envtest")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "..", "test", "data", "crds")},
		ErrorIfCRDPathMissing: true,
		UseExistingCluster:    new(false),
	}
	if dir := getFirstFoundEnvTestBinaryDir(); dir != "" {
		testEnv.BinaryAssetsDirectory = dir
	}
	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())
	DeferCleanup(func() { Expect(testEnv.Stop()).To(Succeed()) })

	By("registering schemes")
	Expect(konfidence.AddToScheme(scheme.Scheme)).To(Succeed())

	By("starting manager")
	startManager()
})

// startManager registers the config and TTL controllers. The execution
// controller is intentionally absent: its specs drive Reconcile directly for
// determinism, and registering it would race the manually driven conditions
// the TTL tests use. The config reconciler is safe to run alongside: the
// execution specs reference sources that do not exist, so it never creates
// promotions for them.
func startManager() {
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:  scheme.Scheme,
		Metrics: metricsserver.Options{BindAddress: "0"},
	})
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())

	Expect(RegisterFieldIndexes(ctx, mgr)).To(Succeed())

	Expect(NewVectorPromotionConfigReconciler(mgr).
		SetupWithManager(mgr)).To(Succeed())

	Expect(NewVectorPromotionTTLReconciler(mgr).
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
