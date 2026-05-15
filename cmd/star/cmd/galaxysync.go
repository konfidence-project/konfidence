package cmd

import (
	"os"

	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	galaxysync "github.com/konfidence-project/konfidence/internal/star/galaxy-sync"
)

var galaxysyncCmd = &cobra.Command{
	Use:   "galaxysync",
	Short: "Run the galaxy-sync controller standalone",
	RunE:  startGalaxySyncController,
}

func startGalaxySyncController(cmd *cobra.Command, args []string) error {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "e83e292c.konfidence.cloud",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		return err
	}

	ctx := ctrl.SetupSignalHandler()

	if err := galaxysync.SetupControllers(mgr, setupLog, scheme, galaxysync.Options{
		ControllerNamespace: os.Getenv("CONTROLLER_NAMESPACE"),
	}); err != nil {
		setupLog.Error(err, "unable to set up galaxy-sync controllers")
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
	rootCmd.AddCommand(galaxysyncCmd)
}
