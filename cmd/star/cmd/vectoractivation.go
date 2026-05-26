//nolint:dupl // TODO(konfidence-project#659): extract shared standalone controller startup wiring.
package cmd

import (
	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	vectoractivation "github.com/konfidence-project/konfidence/internal/star/vector-activation"
)

var vectoractivationCmd = &cobra.Command{
	Use:   "vectoractivation",
	Short: "Run the vector-activation controller standalone",
	RunE:  startVectorActivationController,
}

func startVectorActivationController(cmd *cobra.Command, args []string) error {
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

	if err := vectoractivation.SetupControllers(mgr, setupLog); err != nil {
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
}

func init() {
	vectoractivationCmd.Flags().StringVar(&leaseID, "lease-id", "star-vectoractivation.konfidence.cloud",
		"The ID used for leader election.")
	rootCmd.AddCommand(vectoractivationCmd)
}
