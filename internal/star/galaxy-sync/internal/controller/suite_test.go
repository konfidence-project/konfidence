/*
Copyright 2025.

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
	"time"

	global "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	landscape "github.com/konfidence-project/konfidence/api/star/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	ctx             context.Context
	cancel          context.CancelFunc
	localTestEnv    *envtest.Environment
	remoteTestEnv   *envtest.Environment
	localCfg        *rest.Config
	remoteCfg       *rest.Config
	localK8sClient  client.Client
	remoteK8sClient client.Client
	reconcileScheme *runtime.Scheme
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx, cancel = context.WithCancel(context.TODO())

	SetDefaultEventuallyTimeout(10 * time.Second)

	var err error
	err = global.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = landscape.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	By("bootstrapping local test environment")
	localTestEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "..", "..", "..", "api", "star", "config", "bases", "crd"),
		},
		ErrorIfCRDPathMissing: true,
	}

	// Retrieve the first found binary directory to allow running tests from IDEs
	if getFirstFoundEnvTestBinaryDir() != "" {
		localTestEnv.BinaryAssetsDirectory = getFirstFoundEnvTestBinaryDir()
	}

	// localCfg is defined in this file globally.
	localCfg, err = localTestEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(localCfg).NotTo(BeNil())

	// create local client
	localK8sClient, err = client.New(localCfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(localK8sClient).NotTo(BeNil())

	By("bootstrapping remote test environment")
	remoteTestEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "..", "..", "api", "galaxy", "config", "bases", "crd")},
		ErrorIfCRDPathMissing: true,
	}

	if getFirstFoundEnvTestBinaryDir() != "" {
		remoteTestEnv.BinaryAssetsDirectory = getFirstFoundEnvTestBinaryDir()
	}

	remoteCfg, err = remoteTestEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(remoteCfg).NotTo(BeNil())

	// create remote client (used directly in tests to create/update objects)
	remoteK8sClient, err = client.New(remoteCfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(remoteK8sClient).NotTo(BeNil())

	// create manager
	k8sManager, err := ctrl.NewManager(localCfg, ctrl.Options{
		Scheme: scheme.Scheme,
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
	})
	Expect(err).ToNot(HaveOccurred())

	// build a cluster.Cluster for the remote env and add it to the manager so
	// its cache is started and synced before the controller's informers run.
	remoteCluster, err := cluster.New(remoteCfg, func(o *cluster.Options) {
		o.Scheme = scheme.Scheme
	})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sManager.Add(remoteCluster)).To(Succeed())

	reconcileScheme = k8sManager.GetScheme()

	err = (&StageSyncReconciler{
		LocalClient:   k8sManager.GetClient(),
		RemoteCluster: remoteCluster,
		Scheme:        reconcileScheme,
		Recorder:      events.NewFakeRecorder(32),
		LandscapeName: "test-star",
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()
})

var _ = AfterSuite(func() {
	By("tearing down the local test environment")
	cancel()
	err := localTestEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
	By("tearing down the remote test environment")
	err = remoteTestEnv.Stop()
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
	basePath := filepath.Join("..", "..", "..", "..", "..", "bin", "k8s")
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
