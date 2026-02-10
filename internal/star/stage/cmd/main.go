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

package main

import (
	"flag"
	"os"
	"time"

	sync "github.com/gravitational/sync-controller/controller"
	common "github.com/konfidence-project/crds/api/common/v1alpha1"
	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	"github.com/konfidence-project/landscape-stage-controller/internal/controller"
	"github.com/konfidence-project/landscape-stage-controller/internal/gc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(common.AddToScheme(scheme))
	utilruntime.Must(landscape.AddToScheme(scheme))

	// +kubebuilder:scaffold:scheme
}

// nolint:gocyclo
func main() {
	var enableLeaderElection bool
	var probeAddr string
	var gcpSyncKubeconfig string
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&gcpSyncKubeconfig, "gcp-sync-kubeconfig", "",
		"Path to kubeconfig file to sync from the remote (GCP) cluster. If empty, no sync is performed.")
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
		LeaderElectionID:       "1ca8113e.konfidence.cloud",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	var remoteConfigObj *rest.Config
	{
		if gcpSyncKubeconfig != "" {
			setupLog.Info("Using provided GCP sync kubeconfig for remote cluster", "path", gcpSyncKubeconfig)
			cfg, err := clientcmd.BuildConfigFromFlags("", gcpSyncKubeconfig)
			if err != nil {
				setupLog.Error(err, "unable to build remote config from kubeconfig", "path", gcpSyncKubeconfig)
				os.Exit(1)
			}
			remoteConfigObj = cfg
			remoteCluster, err := cluster.New(remoteConfigObj, func(options *cluster.Options) {
				options.Scheme = scheme
			})
			if err != nil {
				setupLog.Error(err, "unable to create remote cluster")
				os.Exit(1)
			}
			if err := mgr.Add(remoteCluster); err != nil {
				setupLog.Error(err, "unable to add remote cluster to manager")
				os.Exit(1)
			}

			if err := (&sync.Reconciler{
				Client:                 mgr.GetClient(),
				RemoteClient:           remoteCluster.GetClient(),
				RemoteCache:            remoteCluster.GetCache(),
				Scheme:                 mgr.GetScheme(),
				Resource:               &common.Stage{},
				RemoteResourceSuffix:   "",
				LocalNamespaceSuffix:   "",
				NamespacePrefix:        "",
				LocalSecretNames:       []string{},
				LocalPropagationPolicy: client.PropagationPolicy(metav1.DeletePropagationForeground),
				ConcurrentReconciles:   1,
			}).SetupWithManager(mgr); err != nil {
				setupLog.Error(err, "unable to create controller", "controller", "sync-controller")
				os.Exit(1)
			}

			setupLog.Info("setup of sync-controller finished")
		} else {
			setupLog.Info("No GCP sync kubeconfig provided; skipping setup of sync-controller and remote cluster")
		}
	}

	if err := (&controller.StageReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor(controller.StageControllerName),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Stage")
		os.Exit(1)
	}

	if err := (&controller.StageVersionReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor(controller.StageVersionControllerName),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "StageVersion")
		os.Exit(1)
	}

	if err := (&controller.StageVersionUsageReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "StageVersionUsage")
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

	signalContext := ctrl.SetupSignalHandler()
	garbageCollector := &gc.StageVersionGarbageCollector{
		Client:   mgr.GetClient(),
		Interval: 15 * time.Second,
		Recorder: mgr.GetEventRecorderFor(gc.StageVersionGarbageCollectorName),
	}

	setupLog.Info("Starting stageVersion garbage collector")
	go func() {
		if err := garbageCollector.Start(signalContext); err != nil {
			setupLog.Error(err, "An error occurred while starting/running the stageVersion garbage collector")
		}
	}()

	setupLog.Info("starting manager")
	if err := mgr.Start(signalContext); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
