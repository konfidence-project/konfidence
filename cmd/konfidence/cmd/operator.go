package cmd

import (
	"context"
	"fmt"

	"github.com/konfidence-project/konfidence/internal/project"
	"github.com/konfidence-project/konfidence/internal/stage"
	"github.com/konfidence-project/konfidence/internal/stageconfiguration"
	"github.com/konfidence-project/konfidence/internal/taskorchestration"
	"github.com/konfidence-project/konfidence/internal/vectoractivation"
	"github.com/konfidence-project/konfidence/internal/vectorassembly"
	"github.com/konfidence-project/konfidence/internal/vectordeployment"
	"github.com/konfidence-project/konfidence/internal/vectorpromotion"
	pkgcmd "github.com/konfidence-project/konfidence/pkg/cmd"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

func startOperator(cmd *cobra.Command, args []string) error {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: probeAddr,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       leaseID,
	})

	if err != nil {
		setupLog.Error(err, "unable to start manager")
		return err
	}

	signalContext, cancel := context.WithCancel(ctrl.SetupSignalHandler())
	defer cancel()

	controllerSetups := buildControllerSetups(signalContext, cancel, mgr, enableGalaxy)

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
	if err := mgr.Start(signalContext); err != nil {
		setupLog.Error(err, "problem running manager")
		return err
	}

	return nil
}

// buildControllerSetups returns the registry of controller setup closures.
// The galaxy controllers are simply not registered when enableGalaxy is
// false, so FilterEnabledControllers rejects their names as unknown.
func buildControllerSetups(ctx context.Context, cancel context.CancelFunc, mgr manager.Manager, enableGalaxy bool) map[string]func() error {
	limiter := crypto.NewLimiter(0)

	setups := map[string]func() error{
		stage.OperatorFlagName: func() error {
			if err := stage.SetupControllers(mgr, setupLog); err != nil {
				return err
			}
			gc := stage.NewGarbageCollector(mgr)
			setupLog.Info("Starting stageVersion garbage collector")
			go func() {
				if err := gc.Start(ctx); err != nil {
					cancel()
					setupLog.Error(err, "An error occurred while starting/running the stageVersion garbage collector")
				}
			}()
			return nil
		},
		taskorchestration.OperatorFlagName: func() error {
			return taskorchestration.SetupControllers(mgr, setupLog)
		},
		vectoractivation.OperatorFlagName: func() error {
			return vectoractivation.SetupControllers(mgr, setupLog)
		},
		vectordeployment.OperatorFlagName: func() error {
			registrySecret, err := resolveRegistryCredentials(ctx, mgr)
			if err != nil {
				setupLog.Error(err, "unable to load registry credentials secret")
				return err
			}
			return vectordeployment.SetupControllers(ctx, mgr, setupLog, vectordeployment.Options{
				OCISecret: registrySecret,
				Limiter:   limiter,
			})
		},
	}

	if !enableGalaxy {
		return setups
	}

	setups[project.OperatorFlagName] = func() error {
		return project.SetupControllers(mgr, project.Options{})
	}
	setups[stageconfiguration.OperatorFlagName] = func() error {
		return stageconfiguration.SetupControllers(mgr, stageconfiguration.Options{
			Limiter: limiter,
		})
	}
	setups[vectorassembly.OperatorFlagName] = func() error {
		return vectorassembly.SetupControllers(mgr, vectorassembly.Options{
			Limiter: limiter,
		})
	}
	setups[vectorpromotion.OperatorFlagName] = func() error {
		return vectorpromotion.SetupControllers(ctx, mgr, vectorpromotion.Options{
			Limiter: limiter,
		})
	}

	return setups
}

// resolveRegistryCredentials loads the registry credentials secret from the k8s cluster.
// Returns nil if the secret is not found.
func resolveRegistryCredentials(ctx context.Context, mgr manager.Manager) (*corev1.Secret, error) {
	const secretName = "registry-credentials"
	const secretNamespace = "konfidence-system"

	secret := &corev1.Secret{}
	err := mgr.GetAPIReader().Get(ctx, types.NamespacedName{Namespace: secretNamespace, Name: secretName}, secret)
	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get secret %s/%s: %w", secretNamespace, secretName, err)
	}

	return secret, nil
}
