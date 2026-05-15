package cmd

import (
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/multicluster-runtime/pkg/multicluster"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/kcp-dev/multicluster-provider/apiexport"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"

	stageconfiguration "github.com/konfidence-project/konfidence/internal/galaxy/stage-configuration"
	vectorpromotion "github.com/konfidence-project/konfidence/internal/galaxy/vector-promotion"
)

func startOperator(cmd *cobra.Command, args []string) error {
	cfg := ctrl.GetConfigOrDie()
	leaderElectionCfg := cfg
	if kubernetesServiceHost != "" && kubernetesServicePort != 0 {
		inClusterCfg, err := rest.InClusterConfig()
		if err != nil {
			setupLog.Error(err, "unable to get in-cluster config for leader election")
			return err
		}

		leaderElectionCfg = inClusterCfg
	}

	var err error
	var provider multicluster.Provider
	if kcpEndpointSlice != "" {
		provider, err = apiexport.New(cfg, kcpEndpointSlice, apiexport.Options{Scheme: scheme, Log: &setupLog})
		if err != nil {
			setupLog.Error(err, "unable to construct cluster provider")
			return err
		}
	}

	mgr, err := mcmanager.New(cfg, provider, ctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "galaxy-operator.konfidence.cloud",
		LeaderElectionConfig:   leaderElectionCfg,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		return err
	}

	ctx := ctrl.SetupSignalHandler()

	// Set up shared vector verifier
	vectorVerifier, err := getVectorVerifier(ctx, mgr)
	if err != nil {
		return err
	}

	// Set up stage configuration controllers
	if err := stageconfiguration.SetupControllers(mgr, scheme, cfg, stageconfiguration.Options{
		VectorVerifier: vectorVerifier,
	}); err != nil {
		setupLog.Error(err, "unable to set up stage configuration controllers")
		return err
	}

	// Set up vector promotion controllers
	if err := vectorpromotion.SetupControllers(ctx, mgr, scheme, vectorpromotion.Options{
		VectorVerifier: vectorVerifier,
	}); err != nil {
		setupLog.Error(err, "unable to set up vector promotion controllers")
		return err
	}

	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		return err
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		return err
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		return err
	}

	return nil
}
