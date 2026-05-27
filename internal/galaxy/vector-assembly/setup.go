package vectorassembly

import (
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/konfidence-project/konfidence/internal/galaxy/vector-assembly/internal/controller"
	"github.com/konfidence-project/konfidence/internal/galaxy/vector-assembly/internal/controller/domain"
	"github.com/konfidence-project/konfidence/internal/galaxy/vector-assembly/pkg/ocm"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	"github.com/konfidence-project/konfidence/pkg/ocm/repository"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
)

const OperatorFlagName = "VectorAssembly"

// Options configures the vector assembly controllers.
type Options struct {
	// ArtifactVerifier is used to verify artifact signatures.
	// If nil, artifact verification is disabled.
	ArtifactVerifier crypto.Verifier

	// VectorVerifier is used to verify vector signatures (e.g. base vector).
	// If nil, vector verification is disabled.
	VectorVerifier crypto.Verifier

	// VectorSigner is used to sign newly assembled vectors.
	// If nil, vector signing is disabled.
	VectorSigner crypto.Signer
}

// SetupControllers registers all vector assembly controllers with the given manager.
func SetupControllers(mgr mcmanager.Manager, scheme *runtime.Scheme, opts Options) error {
	adapterConfig := []ocm.AdapterOption{
		ocm.WithArtifactVerifier(opts.ArtifactVerifier),
		ocm.WithVectorVerifier(opts.VectorVerifier),
		ocm.WithVectorSigner(opts.VectorSigner),
	}

	if err := (&controller.VectorTemplateReconciler{
		Mgr:                   mgr,
		Scheme:                scheme,
		OcmClientProvider:     repository.DefaultOciClientProvider,
		VectorOcmPortProvider: ocm.NewPortProvider(adapterConfig...),
		VersionGenerator:      domain.TimestampVectorVersionGenerator,
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	return nil
}
