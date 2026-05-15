/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
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

	vectorassembly "github.com/konfidence-project/konfidence/internal/galaxy/vector-assembly"
	"github.com/konfidence-project/konfidence/pkg/cli"
)

var vectorassemblyCmd = &cobra.Command{
	Use:   "vectorassembly",
	Short: "Start the vector assembly controllers",
	Long:  `Starts the VectorTemplate controller for assembling vectors from templates.`,
	RunE:  startVectorAssembly,
}

func startVectorAssembly(cmd *cobra.Command, args []string) error {
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
		LeaderElectionID:       "f4a7b3d1.konfidence.cloud",
		LeaderElectionConfig:   leaderElectionCfg,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		return err
	}

	ctx := ctrl.SetupSignalHandler()

	cryptoCfg, err := cli.ResolveCryptoConfig(ctx, mgr.GetLocalManager(), setupLog)
	if err != nil {
		return err
	}

	if err := vectorassembly.SetupControllers(mgr, scheme, vectorassembly.Options{
		ArtifactVerifier: cryptoCfg.ArtifactVerifier,
		VectorVerifier:   cryptoCfg.VectorVerifier,
		VectorSigner:     cryptoCfg.VectorSigner,
	}); err != nil {
		setupLog.Error(err, "unable to set up vector assembly controllers")
		return err
	}

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

func init() {
	rootCmd.AddCommand(vectorassemblyCmd)
}
