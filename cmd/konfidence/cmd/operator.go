package cmd

import (
	"context"

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
	"github.com/konfidence-project/konfidence/pkg/operator"
	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// controllerDomains lists every controller domain wired into the binary,
// sorted by name.
func controllerDomains() []operator.Domain {
	return []operator.Domain{
		landscape.Domain(),
		project.Domain(),
		stage.Domain(),
		stageconfiguration.Domain(),
		taskorchestration.Domain(),
		vectoractivation.Domain(),
		vectorassembly.Domain(),
		vectordeployment.Domain(),
		vectorpromotion.Domain(),
	}
}

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

	enabled, err := pkgcmd.FilterEnabledControllers(controllersSpec, operator.Names(domains))
	if err != nil {
		setupLog.Error(err, "invalid --controllers flag")
		return err
	}

	deps := operator.Deps{
		Mgr:      mgr,
		Logger:   setupLog,
		Limiter:  crypto.NewLimiter(0),
		Shutdown: cancel,
	}

	for _, domain := range domains {
		if !enabled[domain.Name] {
			setupLog.Info("controller disabled", "controller", domain.Name)
			continue
		}
		setupLog.Info("setting up controller", "controller", domain.Name)
		if err := domain.Setup(signalContext, deps); err != nil {
			setupLog.Error(err, "unable to set up controller", "controller", domain.Name)
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
