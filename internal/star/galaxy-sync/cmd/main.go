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

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	global "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	landscape "github.com/konfidence-project/konfidence/api/star/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/star/galaxy-sync/internal/config"
	"github.com/konfidence-project/konfidence/internal/star/galaxy-sync/internal/controller"
	// +kubebuilder:scaffold:imports
)

// getControllerNamespace returns the namespace the controller is running in by
// reading the CONTROLLER_NAMESPACE environment variable (injected via the
// Downward API). Falls back to "default" when the variable is not set (e.g.
// local development).
func getControllerNamespace() string {
	if ns := os.Getenv("CONTROLLER_NAMESPACE"); ns != "" {
		return ns
	}
	return "default"
}

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(landscape.AddToScheme(scheme))
	utilruntime.Must(global.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

// nolint:gocyclo
func main() {
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "e83e292c.konfidence.cloud",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err := setupStageSyncReconciler(setupLog, mgr); err != nil {
		setupLog.Error(err, "unable to set up StageSyncReconciler")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

// setupStageSyncReconciler wires up the StageSyncReconciler.
//
// It attempts to read the remote kubeconfig from a Secret named
// galaxySyncKubeconfigSecretName in secretNamespace. When the Secret is not
// found the controller falls back to using the local cluster as the remote
// cluster (single-cluster use-case).
func setupStageSyncReconciler(log logr.Logger, mgr ctrl.Manager) error {
	// Use a direct (non-cached) client so the Secret can be fetched before
	// the manager's cache is started.
	directClient, err := client.New(mgr.GetConfig(), client.Options{Scheme: scheme})
	if err != nil {
		return fmt.Errorf("unable to create direct client for kubeconfig Secret lookup: %w", err)
	}

	remoteConfig, err := config.FromSecret(directClient, getControllerNamespace())
	if err != nil {
		return fmt.Errorf("unable to resolve remote kubeconfig: %w", err)
	}

	// remoteCluster is the cluster the controller watches StageSync objects on.
	// Default to the local cluster (single-cluster mode).
	var remoteCluster cluster.Cluster = mgr

	if remoteConfig != nil {
		// Multi-cluster: build a dedicated cluster for the remote (Galaxy) side.
		log.Info("Remote kubeconfig found; running in multi-cluster mode",
			"kubeconfig-secret", fmt.Sprintf("%s/%s", getControllerNamespace(), config.SecretName),
			"secret-key", config.SecretKey,
			"remote-cluster-server", remoteConfig.Host)

		remoteCluster, err = cluster.New(remoteConfig, func(options *cluster.Options) {
			options.Scheme = scheme
			// Restrict the remote cache to only StageSync objects.
			// The remote cluster is a kcp workspace where StageSync is served
			// via an APIBinding — restricting the cache prevents discovery
			// errors caused by trying to watch resource types that are not
			// available on that workspace.
			options.Cache.ByObject = map[client.Object]cache.ByObject{
				&global.StageSync{}: {},
			}
		})
		if err != nil {
			return fmt.Errorf("unable to create remote cluster: %w", err)
		}
		// Adding the cluster to the manager ensures its cache is started and
		// fully synced before the controller's informers are set up.
		if err := mgr.Add(remoteCluster); err != nil {
			return fmt.Errorf("unable to add remote cluster to manager: %w", err)
		}
	} else {
		log.Info("No remote kubeconfig Secret found; running in single-cluster mode")
	}

	if err := (&controller.StageSyncReconciler{
		LocalClient:   mgr.GetClient(),
		RemoteCluster: remoteCluster,
		Scheme:        scheme,
		Recorder:      remoteCluster.GetEventRecorder(controller.StageSyncControllerName),
		LandscapeName: config.LandscapeName(),
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create StageSyncReconciler: %w", err)
	}

	return nil
}
