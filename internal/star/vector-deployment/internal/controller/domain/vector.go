package domain

import (
	"context"
)

//go:generate go run go.uber.org/mock/mockgen -source=vector.go -destination=mocks/mock_ocm_port.go -package=mocks

// VectorOcmPort defines a subset of functionalities from the OCM library.
type VectorOcmPort interface {
	GetVectorByReference(ctx context.Context, vectorReference VectorReference) (*Vector, error)
	GetArtifactManifestByReference(ctx context.Context, ociUrl string, artifactName ArtifactReference) (*ArtifactManifest, error)
}

// Vector represents a domain model for the vector-deployment-controller.
type Vector struct {
	Reference     VectorReference
	ComponentSpec string
	Artifacts     []ArtifactReference
}

type ArtifactReference struct {
	Version       string
	ComponentName string
}

type VectorReference struct {
	OciRegistryUrl string
	Component      string
	Version        string
}

// ArtifactManifest represents the manifest of an artefact.
type ArtifactManifest struct {
	Name       string
	Version    string
	Type       string
	AllowReuse bool
	Tasks      []TaskManifest
	Resources  []OCMResource
}

type OCMResource struct {
	Name    string
	Content []byte
	Type    string
}

type TaskManifest struct {
	Name      string
	Type      string
	DependsOn []string
	Spec      string
}
