package cmd

import (
	"context"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/deploymenttarget"
	"github.com/konfidence-project/konfidence/internal/landscape"
	"github.com/konfidence-project/konfidence/internal/project"
	"github.com/konfidence-project/konfidence/internal/stage"
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
		deploymenttarget.Domain(),
		landscape.Domain(),
		project.Domain(),
		stage.Domain(),
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

	// Build the one process-wide Verifier: ParallelVerifier (limiter) →
	// CachingVerifier (LRU) → OCMVerifier (crypto). Every domain receives
	// this exact instance via deps.Verifier — different reconcilers verifying
	// the same (descriptor, signature) pair share a single cache entry.
	//
	// INVARIANT: this single `limiter` instance is the process-wide budget for
	// ALL CPU-bound crypto. It flows into the verifier here via WithParallelism
	// AND into deps.Limiter below, from where signing domains hand it to their
	// signers (SignerBuilder.WithLimiter). Both draw from the same token pool,
	// so a signing burst cannot oversubscribe the cores verification is using.
	// Do NOT construct a second limiter for the verifier — that would silently
	// split the budget in two.
	limiter := crypto.NewLimiter(0)
	sharedVerifier, err := crypto.NewVerifierBuilder().
		WithParallelism(limiter).
		WithCache(1024, 30*time.Minute).
		WithLogger(setupLog).
		Build()
	if err != nil {
		setupLog.Error(err, "unable to build shared verifier")
		return err
	}

	deps := operator.Deps{
		Mgr:      mgr,
		Logger:   setupLog,
		Limiter:  limiter,
		Verifier: sharedVerifier,
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

		// Project namespace resources
		if err := konfidence.SetupLandscapeWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to set up Landscape webhook")
			return err
		}
		if err := konfidence.SetupVectorTemplateWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to set up VectorTemplate webhook")
			return err
		}
		if err := konfidence.SetupVectorPromotionConfigWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to set up VectorPromotionConfig webhook")
			return err
		}
		// Landscape namespace resources
		if err := konfidence.SetupStageWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to set up Stage webhook")
			return err
		}
		if err := konfidence.SetupDeploymentTargetWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to set up DeploymentTarget webhook")
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
