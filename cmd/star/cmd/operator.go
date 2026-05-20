package cmd

import (
	"context"
	"fmt"
	"os"

	galaxysync "github.com/konfidence-project/konfidence/internal/star/galaxy-sync"
	"github.com/konfidence-project/konfidence/internal/star/stage"
	taskorchestration "github.com/konfidence-project/konfidence/internal/star/task-orchestration"
	vectoractivation "github.com/konfidence-project/konfidence/internal/star/vector-activation"
	vectordeployment "github.com/konfidence-project/konfidence/internal/star/vector-deployment"
	"github.com/konfidence-project/konfidence/internal/star/vector-deployment/pkg/ocm"
	"github.com/konfidence-project/konfidence/pkg/cli"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func startOperator(cmd *cobra.Command, args []string) error {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "1ca8113e.konfidence.cloud",
	})

	if err != nil {
		setupLog.Error(err, "unable to start manager")
		return err
	}

	signalContext, cancel := context.WithCancel(ctrl.SetupSignalHandler())
	defer cancel()

	// Resolve crypto and OcmAdapter for vector-deployment.
	// TODO: resolve lazily based on which controllers are enabled
	cryptoCfg, err := cli.ResolveCryptoConfig(signalContext, mgr, setupLog)
	if err != nil {
		setupLog.Error(err, "unable to resolve crypto config")
		return err
	}

	registrySecret, err := resolveRegistryCredentials(signalContext, mgr)
	if err != nil {
		setupLog.Error(err, "unable to load registry credentials secret")
		return err
	}

	ocmAdapter, err := ocm.NewOcmAdapter(signalContext, registrySecret, cryptoCfg.VectorVerifier, cryptoCfg.ArtifactVerifier)
	if err != nil {
		setupLog.Error(err, "unable to create OCM adapter")
		return err
	}

	setups := []struct {
		Name  string
		Setup func() error
	}{
		{
			Name: "Stage",
			Setup: func() error {
				if err := stage.SetupControllers(mgr, setupLog); err != nil {
					return err
				}
				gc := stage.NewGarbageCollector(mgr)
				setupLog.Info("Starting stageVersion garbage collector")
				go func() {
					if err := gc.Start(signalContext); err != nil {
						cancel()
						setupLog.Error(err, "An error occurred while starting/running the stageVersion garbage collector")
					}
				}()
				return nil
			},
		},
		{
			Name: "TaskOrchestration",
			Setup: func() error {
				return taskorchestration.SetupControllers(mgr, setupLog)
			},
		},
		{
			Name: "VectorActivation",
			Setup: func() error {
				return vectoractivation.SetupControllers(mgr, setupLog)
			},
		},
		{
			Name: "VectorDeployment",
			Setup: func() error {
				return vectordeployment.SetupControllers(mgr, setupLog, vectordeployment.Options{
					OcmAdapter: ocmAdapter,
				})
			},
		},
		{
			Name: "GalaxySync",
			Setup: func() error {
				return galaxysync.SetupControllers(mgr, setupLog, scheme, galaxysync.Options{
					ControllerNamespace: os.Getenv("CONTROLLER_NAMESPACE"),
				})
			},
		},
	}

	names := make([]string, len(setups))
	for i, s := range setups {
		names[i] = s.Name
	}
	enabled, err := cli.Filter(controllersSpec, names)
	if err != nil {
		setupLog.Error(err, "invalid --controllers flag")
		return err
	}

	for _, s := range setups {
		if !enabled[s.Name] {
			setupLog.Info("controller disabled", "controller", s.Name)
			continue
		}
		setupLog.Info("setting up controller", "controller", s.Name)
		if err := s.Setup(); err != nil {
			setupLog.Error(err, "unable to set up controller", "controller", s.Name)
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
	if err := mgr.Start(signalContext); err != nil {
		setupLog.Error(err, "problem running manager")
		return err
	}

	return nil
}

// TODO: the credentials for accessing OCI registries should be configured in a controller-specific configuration.
// resolveRegistryCredentials loads the registry credentials secret from the k8s cluster.
// Returns nil if the secret is not found.
func resolveRegistryCredentials(ctx context.Context, mgr manager.Manager) (*v1.Secret, error) {
	const secretName = "registry-credentials"
	const secretNamespace = "konfidence-system"

	secret := &v1.Secret{}
	err := mgr.GetAPIReader().Get(ctx, types.NamespacedName{Namespace: secretNamespace, Name: secretName}, secret)
	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get secret %s/%s: %w", secretNamespace, secretName, err)
	}

	return secret, nil
}
