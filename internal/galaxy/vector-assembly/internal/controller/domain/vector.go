package domain

//go:generate go run go.uber.org/mock/mockgen -source=vector.go -destination=mocks/mock_ocm_port.go -package=mocks

import (
	"context"

	pkgocm "github.com/konfidence-project/pkg/ocm/repository"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	"ocm.software/open-component-model/bindings/go/runtime"
)

// VectorOcmPort defines an interface for interacting with the OCM repository to manage vectors and their associated artifacts.
// It provides methods to retrieve artifacts and vectors, and create vectors in the OCM repository.
type VectorOcmPort interface {
	// GetArtifacts retrieves the artifacts associated with the given component references from the OCM repository.
	GetArtifacts(ctx context.Context, references []compref.Ref) ([]Artifact, error)

	// GetVector retrieves the vector associated with the given component reference from the OCM repository.
	// It returns ErrVectorNotFound in case the vector was not found.
	GetVector(ctx context.Context, vectorRef compref.Ref) (Vector, error)

	// CreateVector creates the specified vector in the repository specified by repoSpec.
	// After CreateVector returns the vector is retrievable via its alias.
	CreateVector(ctx context.Context, repoSpec runtime.Typed, vector Vector, alias string) error
}

type VectorOcmPortProvider interface {
	NewVectorOcmPort(client pkgocm.Client) VectorOcmPort
}

type VectorOcmPortProviderFunc func(client pkgocm.Client) VectorOcmPort

func (f VectorOcmPortProviderFunc) NewVectorOcmPort(client pkgocm.Client) VectorOcmPort {
	return f(client)
}

var (
	ErrVectorNotFound = pkgocm.ErrNotFound
)

type Vector struct {
	Version   string
	Name      string
	Artifacts []Artifact
}

type Artifact struct {
	Version    string
	Name       string
	Digest     string
	SourceRepo runtime.Typed
}

func HasDrift(desired, actual []Artifact) bool {
	if len(desired) != len(actual) {
		return true
	}

	for _, desiredElement := range desired {
		desiredElementFound := false
		for _, actualElement := range actual {
			if desiredElement.Name == actualElement.Name {
				if desiredElement.Version != actualElement.Version {
					return true
				}
				desiredElementFound = true
				break
			}
		}
		if !desiredElementFound {
			return true
		}
	}
	return false
}
