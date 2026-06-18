package vector

import (
	"bytes"
	"context"

	pkgocm "github.com/konfidence-project/konfidence/pkg/ocm/repository"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	"ocm.software/open-component-model/bindings/go/runtime"
)

// OcmPort defines an interface for interacting with the OCM repository to manage vectors and their associated artifacts.
// It provides methods to retrieve artifacts and vectors, and create vectors in the OCM repository.
type OcmPort interface {
	// GetArtifacts retrieves the artifacts associated with the given component references from the OCM repository.
	GetArtifacts(ctx context.Context, references []compref.Ref) ([]Artifact, error)

	// GetVector retrieves the vector associated with the given component reference from the OCM repository.
	// It returns ErrVectorNotFound in case the vector was not found.
	GetVector(ctx context.Context, vectorRef compref.Ref) (Vector, error)

	// CreateVector creates the specified vector in the repository specified by repoSpec.
	// After CreateVector returns the vector is retrievable via its alias.
	CreateVector(ctx context.Context, repoSpec runtime.Typed, vector Vector, alias string) error
}

type OcmPortProvider interface {
	NewVectorOcmPort(client pkgocm.Client) OcmPort
}

type OcmPortProviderFunc func(client pkgocm.Client) OcmPort

func (f OcmPortProviderFunc) NewVectorOcmPort(client pkgocm.Client) OcmPort {
	return f(client)
}

var (
	ErrVectorNotFound = pkgocm.ErrNotFound
)

type Vector struct {
	Version      string
	Name         string
	Artifacts    []Artifact
	VectorConfig *VectorConfiguration
}

type Artifact struct {
	Version    string
	Name       string
	Digest     string
	SourceRepo runtime.Typed
}

type VectorConfiguration struct {
	Content []byte
}

func HasDrift(currentVector, desiredVector Vector) bool {
	return hasArtifactDrift(currentVector.Artifacts, desiredVector.Artifacts) ||
		hasVectorConfigDrift(currentVector.VectorConfig, desiredVector.VectorConfig)
}

func hasArtifactDrift(currentArtifacts, desiredArtifacts []Artifact) bool {
	if len(desiredArtifacts) != len(currentArtifacts) {
		return true
	}

	for _, desiredArtifact := range desiredArtifacts {
		desiredArtifactFound := false

		for _, currentArtifact := range currentArtifacts {
			if desiredArtifact.Name == currentArtifact.Name {
				if desiredArtifact.Version != currentArtifact.Version {
					return true
				}
				desiredArtifactFound = true
				break
			}
		}

		if !desiredArtifactFound {
			return true
		}
	}

	return false
}

func hasVectorConfigDrift(currentVectorConfig, desiredVectorConfig *VectorConfiguration) bool {
	if desiredVectorConfig == nil && currentVectorConfig == nil {
		return false
	}

	if desiredVectorConfig != nil && desiredVectorConfig.Content != nil &&
		currentVectorConfig != nil && currentVectorConfig.Content != nil {
		return !bytes.Equal(desiredVectorConfig.Content, currentVectorConfig.Content)
	}

	return true
}
