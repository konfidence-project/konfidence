package assembly

//go:generate go run go.uber.org/mock/mockgen -destination=mocks/mock_vector_repository.go -package=mocks github.com/konfidence-project/konfidence/internal/galaxy/vectorassembly/assembly VectorRepository

import (
	"context"

	galaxy "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/galaxy/vectorassembly/vector"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	"ocm.software/open-component-model/bindings/go/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ErrVectorNotFound is returned when a requested vector cannot be found in the OCM repository.
// It re-exports the domain sentinel so callers can match it via errors.Is without depending on
// the concrete adapter implementation.
var ErrVectorNotFound = vector.ErrVectorNotFound

// VectorRepository stores and retrieves vectors and their artifacts in an OCM repository.
//
// OCM itself only understands components and descriptors. This interface is konfidence's higher-level
// view on top of it: callers work with vectors and artifacts, while the implementation does the
// translation to and from OCM components and descriptors. It provides methods to retrieve artifacts
// and vectors, and to create vectors in the OCM repository.
type VectorRepository interface {
	// GetArtifacts retrieves the artifacts associated with the given component references from the OCM repository.
	GetArtifacts(ctx context.Context, references []compref.Ref) ([]vector.Artifact, error)

	// GetVector retrieves the vector associated with the given component reference from the OCM repository.
	// It returns ErrVectorNotFound in case the vector was not found.
	GetVector(ctx context.Context, vectorRef compref.Ref) (vector.Vector, error)

	// CreateVector creates the specified vector in the repository specified by repoSpec.
	// After CreateVector returns the vector is retrievable via its alias.
	CreateVector(ctx context.Context, repoSpec runtime.Typed, v vector.Vector, alias string) error
}

// VectorRepositoryProvider builds a VectorRepository for a single reconcile, resolving and configuring
// the underlying OCM client from the supplied Kubernetes credentials configuration. Callers pass the
// reconcile dependencies and receive a ready-to-use VectorRepository; the OCM client is owned by the
// returned repository.
type VectorRepositoryProvider interface {
	VectorRepositoryFor(ctx context.Context, k8sClient client.Reader, namespace string, credentialsConfig []galaxy.CredentialsConfig) (VectorRepository, error)
}

// VectorRepositoryProviderFunc lets a plain function be used as a VectorRepositoryProvider.
type VectorRepositoryProviderFunc func(
	ctx context.Context,
	k8sClient client.Reader,
	namespace string,
	credentialsConfig []galaxy.CredentialsConfig,
) (VectorRepository, error)

func (f VectorRepositoryProviderFunc) VectorRepositoryFor(
	ctx context.Context,
	k8sClient client.Reader,
	namespace string,
	credentialsConfig []galaxy.CredentialsConfig,
) (VectorRepository, error) {
	return f(ctx, k8sClient, namespace, credentialsConfig)
}
