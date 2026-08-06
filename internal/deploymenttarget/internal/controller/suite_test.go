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
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var (
	testEnv *envtest.Environment
	cfg     *rest.Config
)

func TestController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DeploymentTarget Controller Suite")
}

var _ = BeforeSuite(func() {
	Expect(konfidence.AddToScheme(scheme.Scheme)).To(Succeed())
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "..", "test", "data", "crds")},
		ErrorIfCRDPathMissing: true,
	}
	if assets := firstEnvtestAssets(); assets != "" {
		testEnv.BinaryAssetsDirectory = assets
	}
	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	Expect(testEnv.Stop()).To(Succeed())
})

func startManager() (client.Client, context.CancelFunc) {
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{Scheme: scheme.Scheme, Metrics: metricsserver.Options{BindAddress: "0"}})
	Expect(err).NotTo(HaveOccurred())
	Expect(NewReconciler(mgr).SetupWithManager(mgr)).To(Succeed())
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer GinkgoRecover()
		Expect(mgr.Start(ctx)).To(Succeed())
	}()
	Eventually(func() bool { return mgr.GetCache().WaitForCacheSync(ctx) }).Should(BeTrue())
	return mgr.GetClient(), cancel
}

func firstEnvtestAssets() string {
	base := filepath.Join("..", "..", "..", "..", "bin", "k8s")
	entries, err := os.ReadDir(base)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return filepath.Join(base, entry.Name())
		}
	}
	return ""
}
