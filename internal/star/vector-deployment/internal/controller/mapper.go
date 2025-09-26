package controller

import (
	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/json"
	"ocm.software/ocm/api/ocm"
	"ocm.software/ocm/api/ocm/compdesc"

	"github.com/konfidence-project/landscape-vector-deployment-controller/internal/controller/domain"
)

func mapVectorDeploymentToDomain(vectorDeployment landscape.VectorDeployment) (*domain.Vector, error) {
	ocmRef, err := parseComponentVersionUrl(vectorDeployment.Spec.Vector)
	if err != nil {
		return nil, err
	}

	if !ocmRef.IsVersion() {
		return nil, errors.Errorf("vector reference %q is not a version", vectorDeployment.Spec.Vector)
	}

	var artifacts []domain.ArtifactReference
	// validate that the ResolvedVectorOcm is a valid ComponentSpec JSON
	if vectorDeployment.Status.ResolvedVectorOcm != "" {
		var cs compdesc.ComponentSpec
		err = json.Unmarshal([]byte(vectorDeployment.Status.ResolvedVectorOcm), &cs)
		if err != nil {
			// todo: other solutions for returning the error, e.g.
			// - 1. instead of an error, we could return a domain error and fetch the component spec from OCM again
			// - 2. adapter could log a warning and set a "" for ComponentSpec, the Reconciler will fetch it from OCM again
			return nil, errors.Wrapf(err, "failed to unmarshal component spec")
		}

		artifacts = make([]domain.ArtifactReference, len(cs.References))
		for i, ref := range cs.References {
			artifacts[i] = domain.ArtifactReference{
				Version:       ref.Version,
				ComponentName: ref.ComponentName,
			}
		}
	}

	vector := domain.Vector{
		Reference: domain.VectorReference{
			OciRegistryUrl: vectorDeployment.Spec.Vector,
			Component:      ocmRef.Component,
			Version:        *ocmRef.Version,
		},
		ComponentSpec: vectorDeployment.Status.ResolvedVectorOcm,
		Artifacts:     artifacts,
	}

	return &vector, nil
}

func parseComponentVersionUrl(ref string) (*ocm.RefSpec, error) {
	ocmRef, err := ocm.ParseRef(ref)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid vector reference %q", ref)
	}
	return &ocmRef, nil
}
