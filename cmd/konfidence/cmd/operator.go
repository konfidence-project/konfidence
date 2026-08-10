package cmd

import (
	"context"
	"fmt"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/landscape"
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
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

func startOperator(_ *cobra.Command, _ []string) error {
	mgrOptions := ctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: probeAddr,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       leaseID,
	}

	if enableWebhooks {
		mgrOptions.WebhookServer = webhook.NewServer(webhook.Options{
			CertDir: webhookCertDir,
			Port:    webhookPort,
		})
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), mgrOptions)

	if err != nil {
		setupLog.Error(err, "unable to start manager")
		return err
	}

	signalContext, cancel := context.WithCancel(ctrl.SetupSignalHandler())
	defer cancel()

	domains := controllerDomains()

	enabled, err := pkgcmd.FilterEnabledControllers(controllersSpec, domainNames(domains))
	if err != nil {
		setupLog.Error(err, "invalid --controllers flag")
		return err
	}

	deps := operatorDeps{
		ctx:     signalContext,
		cancel:  cancel,
		mgr:     mgr,
		limiter: crypto.NewLimiter(0),
	}

	for _, domain := range domains {
		if !enabled[domain.name] {
			setupLog.Info("controller disabled", "controller", domain.name)
			continue
		}
		setupLog.Info("setting up controller", "controller", domain.name)
		if err := domain.setup(deps); err != nil {
			setupLog.Error(err, "unable to set up controller", "controller", domain.name)
			return err
		}
	}

	// +kubebuilder:scaffold:builder

	if enableWebhooks {
		setupLog.Info("setting up webhooks")
		if err := konfidence.SetupLandscapeWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to set up Landscape webhook")
			return err
		}
	} else {
		setupLog.Info("webhooks disabled")
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
	if err := mgr.Start(signalContext); err != nil {
		setupLog.Error(err, "problem running manager")
		return err
	}

	return nil
}

// operatorDeps carries the runtime dependencies injected into a controller
// domain's setup once it is enabled.
type operatorDeps struct {
	ctx     context.Context
	cancel  context.CancelFunc
	mgr     manager.Manager
	limiter crypto.Limiter
}

// controllerDomain is one --controllers toggle: the domain flag name and the
// controllers it runs, as listed in the flag help.
type controllerDomain struct {
	name        string
	controllers string
	setup       func(operatorDeps) error
}

// controllerDomains is the --controllers registry, sorted by name. It is
// dependency-free so the flag help can render it at init time.
func controllerDomains() []controllerDomain {
	return []controllerDomain{
		{landscape.OperatorFlagName, "Landscape", func(d operatorDeps) error {
			return landscape.SetupControllers(d.mgr, landscape.Options{})
		}},
		{project.OperatorFlagName, "Project", func(d operatorDeps) error {
			return project.SetupControllers(d.mgr, project.Options{})
		}},
		{stage.OperatorFlagName,
			"Stage, StageVersion, StageVersionUsage, stageVersion garbage collector",
			func(d operatorDeps) error {
				if err := stage.SetupControllers(d.mgr, setupLog); err != nil {
					return err
				}
				gc := stage.NewGarbageCollector(d.mgr)
				setupLog.Info("Starting stageVersion garbage collector")
				go func() {
					if err := gc.Start(d.ctx); err != nil {
						d.cancel()
						setupLog.Error(err, "An error occurred while starting/running the stageVersion garbage collector")
					}
				}()
				return nil
			}},
		{stageconfiguration.OperatorFlagName, "StageConfiguration", func(d operatorDeps) error {
			return stageconfiguration.SetupControllers(d.mgr, stageconfiguration.Options{
				Limiter: d.limiter,
			})
		}},
		{taskorchestration.OperatorFlagName, "TaskOrchestration", func(d operatorDeps) error {
			return taskorchestration.SetupControllers(d.mgr, setupLog)
		}},
		{vectoractivation.OperatorFlagName, "VectorActivation", func(d operatorDeps) error {
			return vectoractivation.SetupControllers(d.mgr, setupLog)
		}},
		{vectorassembly.OperatorFlagName, "VectorTemplate", func(d operatorDeps) error {
			return vectorassembly.SetupControllers(d.mgr, vectorassembly.Options{
				Limiter: d.limiter,
			})
		}},
		{vectordeployment.OperatorFlagName, "VectorDeployment", func(d operatorDeps) error {
			registrySecret, err := resolveRegistryCredentials(d.ctx, d.mgr)
			if err != nil {
				setupLog.Error(err, "unable to load registry credentials secret")
				return err
			}
			return vectordeployment.SetupControllers(d.ctx, d.mgr, setupLog, vectordeployment.Options{
				OCISecret: registrySecret,
				Limiter:   d.limiter,
			})
		}},
		{vectorpromotion.OperatorFlagName,
			"VectorPromotion, VectorPromotionConfig, VectorPromotion TTL",
			func(d operatorDeps) error {
				return vectorpromotion.SetupControllers(d.ctx, d.mgr, vectorpromotion.Options{
					Limiter: d.limiter,
				})
			}},
	}
}

func domainNames(domains []controllerDomain) []string {
	names := make([]string, len(domains))
	for i, domain := range domains {
		names[i] = domain.name
	}
	return names
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
