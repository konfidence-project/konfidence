/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/multicluster-runtime/pkg/multicluster"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/kcp-dev/multicluster-provider/apiexport"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"

	stageconfiguration "github.com/konfidence-project/konfidence/internal/galaxy/stage-configuration"
	// +kubebuilder:scaffold:imports
)

// stageconfigurationCmd represents the stageconfiguration command
var stageconfigurationCmd = &cobra.Command{
	Use:   "stageconfiguration",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {

		cfg := ctrl.GetConfigOrDie()
		leaderElectionCfg := cfg
		if kubernetesServiceHost != "" && kubernetesServicePort != 0 {
			inClusterCfg, err := rest.InClusterConfig()
			if err != nil {
				setupLog.Error(err, "unable to get in-cluster config for leader election")
				os.Exit(1)
			}

			leaderElectionCfg = inClusterCfg
		}

		var err error
		var provider multicluster.Provider = nil
		if kcpEndpointSlice != "" {
			provider, err = apiexport.New(cfg, kcpEndpointSlice, apiexport.Options{Scheme: scheme, Log: &setupLog})
			if err != nil {
				setupLog.Error(err, "unable to construct cluster provider")
				os.Exit(1)
			}
		}

		mgr, err := mcmanager.New(cfg, provider, ctrl.Options{
			Scheme:                 scheme,
			HealthProbeBindAddress: probeAddr,
			LeaderElection:         enableLeaderElection,
			LeaderElectionID:       "969492ce.konfidence.cloud",
			LeaderElectionConfig:   leaderElectionCfg,
		})
		if err != nil {
			setupLog.Error(err, "unable to start manager")
			os.Exit(1)
		}

		ctx := ctrl.SetupSignalHandler()

		vectorVerifier, err := getVectorVerifier(ctx, mgr)
		if err != nil {
			os.Exit(1)
		}
		if err := stageconfiguration.NewStageConfigurationReconciler(
			mgr,
			scheme,
			cfg,
			vectorVerifier,
		).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "StageConfiguration")
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
		if err := mgr.Start(ctx); err != nil {
			setupLog.Error(err, "problem running manager")
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(stageconfigurationCmd)
}
