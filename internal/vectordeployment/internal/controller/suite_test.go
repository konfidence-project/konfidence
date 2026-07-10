package controller_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	star "github.com/konfidence-project/konfidence/api/star/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var (
	ctx        context.Context
	cancel     context.CancelFunc
	testEnv    *envtest.Environment
	cfg        *rest.Config
	k8sClient  client.Client
	k8sManager manager.Manager
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	var err error

	By("bootstrapping test environment")
	useExternalCluster := false
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "..", "test", "data", "crds", "star")},
		ErrorIfCRDPathMissing: true,
		UseExistingCluster:    &useExternalCluster,
	}
	if dir := getFirstFoundEnvTestBinaryDir(); dir != "" {
		testEnv.BinaryAssetsDirectory = dir
	}
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	k8sManager, err = manager.New(cfg, manager.Options{
		Metrics: metricsserver.Options{BindAddress: "0"},
		Cache:   cache.Options{DefaultNamespaces: map[string]cache.Config{"default": {}}},
	})
	Expect(err).ToNot(HaveOccurred())

	Expect(star.AddToScheme(k8sManager.GetScheme())).To(Succeed())

	k8sClient, err = client.New(cfg, client.Options{Scheme: k8sManager.GetScheme()})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	go func() {
		defer GinkgoRecover()
		Expect(k8sManager.Start(ctx)).ToNot(HaveOccurred(), "failed to run k8s manager")
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
