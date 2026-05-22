package cmd

import (
	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	vectordeployment "github.com/konfidence-project/konfidence/internal/star/vector-deployment"
	"github.com/konfidence-project/konfidence/internal/star/vector-deployment/pkg/ocm"
	pkgcmd "github.com/konfidence-project/konfidence/pkg/cmd"
)

var vectordeploymentCmd = &cobra.Command{
	Use:   "vectordeployment",
	Short: "Run the vector-deployment controller standalone",
	RunE:  startVectorDeploymentController,
}

func startVectorDeploymentController(cmd *cobra.Command, args []string) error {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "a67b73e3.konfidence.cloud",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		return err
	}

	ctx := ctrl.SetupSignalHandler()

	cryptoCfg, err := pkgcmd.ResolveCryptoConfig(ctx, mgr, setupLog)
	if err != nil {
		setupLog.Error(err, "unable to resolve crypto config")
		return err
	}

	registrySecret, err := resolveRegistryCredentials(ctx, mgr)
	if err != nil {
		setupLog.Error(err, "unable to load registry credentials secret")
		return err
	}

	ocmAdapter, err := ocm.NewOcmAdapter(ctx, registrySecret, cryptoCfg.VectorVerifier, cryptoCfg.ArtifactVerifier)
	if err != nil {
		setupLog.Error(err, "unable to create OCM adapter")
		return err
	}

	if err := vectordeployment.SetupControllers(mgr, setupLog, vectordeployment.Options{
		OcmAdapter: ocmAdapter,
	}); err != nil {
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
	rootCmd.AddCommand(vectordeploymentCmd)
}
