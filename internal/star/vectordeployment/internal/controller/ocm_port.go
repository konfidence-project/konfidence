package controller

import (
	"context"

	"ocm.software/open-component-model/bindings/go/oci/compref"
)

//go:generate go run go.uber.org/mock/mockgen -destination=./mocks/mock_ocm_port.go -package=mocks github.com/konfidence-project/konfidence/internal/star/vectordeployment/internal/controller VectorOcmPort

// VectorOcmPort defines a subset of functionalities from the OCM library.
type VectorOcmPort interface {
	GetVectorDescriptor(ctx context.Context, ref compref.Ref) (VectorDescriptor, error)
	GetArtifactManifestByReference(ctx context.Context, ref compref.Ref) (ArtifactManifest, error)
}

type VectorDescriptor struct {
	References     []compref.Ref
	DescriptorJSON []byte
}

type ArtifactManifest struct {
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
