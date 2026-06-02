package ports

import (
	"context"
	"errors"

	"github.com/konfidence-project/konfidence/internal/galaxy/vectorassembly/internal/vector"
	pkgocm "github.com/konfidence-project/konfidence/pkg/ocm/repository"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	"ocm.software/open-component-model/bindings/go/runtime"
)

var ErrVectorNotFound = errors.New("vector not found")

// OcmPort defines an interface for interacting with the OCM repository to manage vectors and their associated artifacts.
// It provides methods to retrieve artifacts and vectors, and create vectors in the OCM repository.
type OcmPort interface {
	// GetArtifacts retrieves the artifacts associated with the given component references from the OCM repository.
	GetArtifacts(ctx context.Context, references []compref.Ref) ([]vector.Artifact, error)

	// GetVector retrieves the vector associated with the given component reference from the OCM repository.
	// It returns ErrVectorNotFound in case the vector was not found.
	GetVector(ctx context.Context, vectorRef compref.Ref) (vector.Vector, error)

	// CreateVector creates the specified vector in the repository specified by repoSpec.
	// After CreateVector returns the vector is retrievable via its alias.
	CreateVector(ctx context.Context, repoSpec runtime.Typed, v vector.Vector, alias string) error
}

type OcmPortProvider interface {
	NewVectorOcmPort(client pkgocm.Client) OcmPort
}

type OcmPortProviderFunc func(client pkgocm.Client) OcmPort

func (f OcmPortProviderFunc) NewVectorOcmPort(client pkgocm.Client) OcmPort {
	return f(client)
}
