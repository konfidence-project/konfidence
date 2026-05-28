package vectordeployment

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/konfidence-project/konfidence/internal/star/vectordeployment/internal/controller"
	"github.com/konfidence-project/konfidence/internal/star/vectordeployment/internal/ocm"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const OperatorFlagName = "VectorDeployment"

// Options configures the vector deployment controllers.
type Options struct {
	// RegistrySecret holds dockerconfigjson-style credentials used to access OCI registries.
	// May be nil if no authentication is required.
	RegistrySecret *v1.Secret

	// VectorVerifier is used to verify vector descriptor signatures.
	// If nil, vector verification is disabled.
	VectorVerifier crypto.Verifier

	// ArtifactVerifier is used to verify artifact descriptor signatures.
	// If nil, artifact verification is disabled.
	ArtifactVerifier crypto.Verifier
}

// SetupControllers registers all vector deployment controllers with the given manager.
func SetupControllers(ctx context.Context, mgr manager.Manager, logger logr.Logger, opts Options) error {
	ocmAdapter, err := ocm.NewAdapter(ctx, opts.RegistrySecret, opts.VectorVerifier, opts.ArtifactVerifier)
	if err != nil {
		return fmt.Errorf("unable to create OCM adapter: %w", err)
	}

	if err := (&controller.VectorDeploymentReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		Recorder:   mgr.GetEventRecorder(controller.VectorDeploymentControllerName),
		OcmAdapter: ocmAdapter,
	}).SetupWithManager(mgr, "vectordeployment"); err != nil {
		logger.Error(err, "unable to create controller", "controller", "VectorDeployment")
		return err
	}
	return nil
}
