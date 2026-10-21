package deployer

import (
	"github.com/konfidence-project/konfidence/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// ArtifactTypePredicate accepts objects that match artifactType.
func ArtifactTypePredicate(artifactType v1alpha1.ArtifactType) predicate.Predicate {
	return predicate.NewPredicateFuncs(func(obj client.Object) bool {
		resource, ok := obj.(v1alpha1.ArtifactTypeProvider)
		return ok && resource.ArtifactType() == artifactType
	})
}
