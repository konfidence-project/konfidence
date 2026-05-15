package cmd

import (
	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	taskorchestration "github.com/konfidence-project/konfidence/internal/star/task-orchestration"
)

var taskorchestrationCmd = &cobra.Command{
	Use:   "taskorchestration",
	Short: "Run the task-orchestration controller standalone",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
			Scheme:                 scheme,
			HealthProbeBindAddress: probeAddr,
			LeaderElection:         enableLeaderElection,
			LeaderElectionID:       "981612b5.konfidence.cloud",
		})
		if err != nil {
			setupLog.Error(err, "unable to start manager")
			return err
		}

		if err := taskorchestration.SetupControllers(mgr, setupLog); err != nil {
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
		if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
			setupLog.Error(err, "problem running manager")
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(taskorchestrationCmd)
}
