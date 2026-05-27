package stageconfiguration

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"

	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
)

const OperatorFlagName = "StageConfiguration"

// Options configures the stage configuration controllers.
type Options struct {
	// VectorVerifier is used to verify vector signatures.
	// If nil or a NoopVerifier, verification is disabled.
	VectorVerifier crypto.Verifier
}

// SetupControllers registers all stage configuration controllers with the given manager.
func SetupControllers(mgr mcmanager.Manager, scheme *runtime.Scheme, restConfig *rest.Config, opts Options) error {
	if err := NewStageConfigurationReconciler(
		mgr,
		scheme,
		restConfig,
		opts.VectorVerifier,
	).SetupWithManager(mgr); err != nil {
		return err
	}

	return nil
}
