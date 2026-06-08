package vectorassembly

import (
	"context"
	"fmt"

	galaxy "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/galaxy/vectorassembly/assembly"
	"github.com/konfidence-project/konfidence/internal/galaxy/vectorassembly/internal/controller"
	"github.com/konfidence-project/konfidence/pkg/ocm"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	"github.com/konfidence-project/konfidence/pkg/ocm/repository"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
)

const OperatorFlagName = "VectorAssembly"

// "Compile-time check: the OCM adapter satisfies VectorRepository."
var _ assembly.VectorRepository = (*ocm.Adapter)(nil)

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
	repositoryProvider := NewVectorRepositoryProvider(
		repository.DefaultOciClientProvider,
		ocm.WithArtifactVerifier(opts.ArtifactVerifier),
		ocm.WithVectorVerifier(opts.VectorVerifier),
		ocm.WithVectorSigner(opts.VectorSigner),
	)

	if err := controller.NewVectorTemplateReconciler(
		mgr,
		scheme,
		repositoryProvider,
	).SetupWithManager(mgr); err != nil {
		return err
	}

	return nil
}

// NewVectorRepositoryProvider returns a VectorRepositoryProvider that, per reconcile, builds an OCM
// client via the given ClientProvider and wraps it in a pkg/ocm Adapter configured with the
// supplied options.
func NewVectorRepositoryProvider(clientProvider repository.ClientProvider, opts ...ocm.AdapterOption) assembly.VectorRepositoryProvider {
	return assembly.VectorRepositoryProviderFunc(func(
		ctx context.Context,
		k8sClient client.Reader,
		namespace string,
		credentialsConfig []galaxy.CredentialsConfig,
	) (assembly.VectorRepository, error) {
		ocmClient, err := clientProvider.NewClient(ctx, k8sClient, namespace, credentialsConfig)
		if err != nil {
			return nil, fmt.Errorf("unable to create OCM client: %w", err)
		}

		adapterOpts := make([]ocm.AdapterOption, 0, len(opts)+1)
		adapterOpts = append(adapterOpts, opts...)
		adapterOpts = append(adapterOpts, ocm.WithClient(ocmClient))
		return ocm.NewAdapter(adapterOpts...), nil
	})
}
