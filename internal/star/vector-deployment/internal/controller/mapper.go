package controller

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/json"
	descv2 "ocm.software/open-component-model/bindings/go/descriptor/v2"
	"ocm.software/open-component-model/bindings/go/oci/compref"
)

// artifactRefsFromStatus parses the artifact component references from the cached
// VectorDescriptor JSON stored in the VectorDeployment status.
func artifactRefsFromStatus(descriptorJSON string, vectorRef compref.Ref) ([]compref.Ref, error) {
	var desc descv2.Descriptor
	if err := json.Unmarshal([]byte(descriptorJSON), &desc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal component spec: %w", err)
	}

	refs := make([]compref.Ref, len(desc.Component.References))
	for i, ref := range desc.Component.References {
		refs[i] = compref.Ref{
			Repository: vectorRef.Repository,
			Component:  ref.Component,
			Version:    ref.Version,
		}
	}
	return refs, nil
}
