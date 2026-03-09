package domain

//go:generate go run go.uber.org/mock/mockgen -source=vector.go -destination=mocks/mock_ocm_port.go -package=mocks

import (
	"context"

	"ocm.software/open-component-model/bindings/go/oci/compref"
	"ocm.software/open-component-model/bindings/go/runtime"
)

// VectorOcmPort defines the interface for interacting with the OCM repository for vector operations.
type VectorOcmPort interface {
	// GetLatestArtifactVersions resolves the versions of the given components in the OCM repository.
	GetLatestArtifactVersions(ctx context.Context, references []compref.Ref) ([]Artifact, error)

	// GetLatestVector retrieves the latest vector from the OCM repository.
	GetLatestVector(ctx context.Context, vectorRef compref.Ref) (Vector, error)

	// CreateVector creates a new vector in the OCM repository.
	CreateVector(ctx context.Context, repoSpec runtime.Typed, vector Vector) error
}

type Vector struct {
	Version   string
	Name      string
	Artifacts []Artifact
}

type Artifact struct {
	Version string
	Name    string
	Digest  string
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
