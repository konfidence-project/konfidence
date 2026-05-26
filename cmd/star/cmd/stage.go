package cmd

import (
	"context"

	"github.com/konfidence-project/konfidence/internal/star/stage"
	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

var stageCmd = &cobra.Command{
	Use:   "stage",
	Short: "Run the stage controllers standalone",
	RunE:  startStageController,
}

func startStageController(cmd *cobra.Command, args []string) error {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       leaseID,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		return err
	}

	if err := stage.SetupControllers(mgr, setupLog); err != nil {
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

	signalContext, cancel := context.WithCancel(ctrl.SetupSignalHandler())
	defer cancel()

	gc := stage.NewGarbageCollector(mgr)
	setupLog.Info("Starting stageVersion garbage collector")
	go func() {
		if err := gc.Start(signalContext); err != nil {
			cancel()
			setupLog.Error(err, "An error occurred while starting/running the stageVersion garbage collector")
		}
	}()

	setupLog.Info("starting manager")
	if err := mgr.Start(signalContext); err != nil {
		setupLog.Error(err, "problem running manager")
		return err
	}
	return nil
}

func init() {
	stageCmd.Flags().StringVar(&leaseID, "lease-id", "star-stage.konfidence.cloud",
		"The ID used for leader election.")
	rootCmd.AddCommand(stageCmd)
}
