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
	"os"
	"path/filepath"
	"testing"

	global "github.com/konfidence-project/crds/api/global/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var (
	ctx       context.Context
	cancel    context.CancelFunc
	testEnv   *envtest.Environment
	cfg       *rest.Config
	k8sClient client.Client
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
		// CRDs are located in the ../../test/data/generated/crds directory
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "test", "data", "generated", "crds")},
		ErrorIfCRDPathMissing: true,
		UseExistingCluster:    &useExternalCluster,
	}

	// Retrieve the first found binary directory to allow running tests from IDEs
	if getFirstFoundEnvTestBinaryDir() != "" {
		testEnv.BinaryAssetsDirectory = getFirstFoundEnvTestBinaryDir()
	}

	// cfg is defined in this file globally.
	// start k8s api server with the defined crds
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

// getFirstFoundEnvTestBinaryDir locates the first binary in the specified path.
// ENVTEST-based tests depend on specific binaries, usually located in paths set by
// controller-runtime. When running tests directly (e.g., via an IDE) without using
// Makefile targets, the 'BinaryAssetsDirectory' must be explicitly configured.
//
// This function streamlines the process by finding the required binaries, similar to
// setting the 'KUBEBUILDER_ASSETS' environment variable. To ensure the binaries are
// properly set up, run 'make setup-envtest' beforehand.
func getFirstFoundEnvTestBinaryDir() string {
	basePath := filepath.Join("..", "..", "bin", "k8s")
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

func StartTestManagerWithReconciler(setup func(ctrl.Manager) error) (client.Client, context.CancelFunc) {
	// set up the k8s manager
	k8sManager, err := manager.New(cfg, manager.Options{})
	Expect(err).ToNot(HaveOccurred())

	// register the globalv1alpha1 scheme
	err = global.AddToScheme(k8sManager.GetScheme())
	Expect(err).NotTo(HaveOccurred())

	// set up the k8s client
	k8sClient, err = client.New(cfg, client.Options{Scheme: k8sManager.GetScheme()})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// register only one reconciler
	Expect(setup(k8sManager)).To(Succeed())

	ctx, cancel := context.WithCancel(context.TODO())
	go func() {
		defer GinkgoRecover()
		Expect(k8sManager.Start(ctx)).To(Succeed())
	}()

	// Wait for cache sync
	Eventually(func() bool {
		return k8sManager.GetCache().WaitForCacheSync(ctx)
	}).Should(BeTrue())

	return k8sClient, cancel
}
