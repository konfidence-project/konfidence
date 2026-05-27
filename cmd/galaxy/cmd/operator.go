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

	stageconfiguration "github.com/konfidence-project/konfidence/internal/galaxy/stageconfiguration"
	vectorassembly "github.com/konfidence-project/konfidence/internal/galaxy/vectorassembly"
	vectorpromotion "github.com/konfidence-project/konfidence/internal/galaxy/vectorpromotion"
	pkgcmd "github.com/konfidence-project/konfidence/pkg/cmd"
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
		LeaderElectionID:       leaseID,
		LeaderElectionConfig:   leaderElectionCfg,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		return err
	}

	ctx := ctrl.SetupSignalHandler()

	// Resolve all crypto dependencies from env vars
	cryptoCfg, err := pkgcmd.ResolveCryptoConfig(ctx, mgr.GetLocalManager(), setupLog)
	if err != nil {
		setupLog.Error(err, "unable to resolve crypto configuration")
		return err
	}

	controllerSetups := map[string]func() error{
		stageconfiguration.OperatorFlagName: func() error {
			return stageconfiguration.SetupControllers(mgr, scheme, cfg, stageconfiguration.Options{
				VectorVerifier: cryptoCfg.VectorVerifier,
			})
		},
		vectorassembly.OperatorFlagName: func() error {
			return vectorassembly.SetupControllers(mgr, scheme, vectorassembly.Options{
				ArtifactVerifier: cryptoCfg.ArtifactVerifier,
				VectorVerifier:   cryptoCfg.VectorVerifier,
				VectorSigner:     cryptoCfg.VectorSigner,
			})
		},
		vectorpromotion.OperatorFlagName: func() error {
			return vectorpromotion.SetupControllers(ctx, mgr, scheme, vectorpromotion.Options{
				VectorVerifier: cryptoCfg.VectorVerifier,
			})
		},
	}

	enabled, err := pkgcmd.FilterEnabledControllers(controllersSpec, controllerSetups)
	if err != nil {
		setupLog.Error(err, "invalid --controllers flag")
		return err
	}

	for name, setup := range controllerSetups {
		if !enabled[name] {
			setupLog.Info("controller disabled", "controller", name)
			continue
		}
		setupLog.Info("setting up controller", "controller", name)
		if err := setup(); err != nil {
			setupLog.Error(err, "unable to set up controller", "controller", name)
			return err
		}
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
