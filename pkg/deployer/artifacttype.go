package deployer

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// ArtifactTypeMatcher is implemented by resources selected by artifact type.
type ArtifactTypeMatcher interface {
	MatchesArtifactType(artifactType string) bool
}

// ArtifactTypePredicate accepts objects that match artifactType.
func ArtifactTypePredicate(artifactType string) predicate.Predicate {
	return predicate.NewPredicateFuncs(func(obj client.Object) bool {
		matcher, ok := obj.(ArtifactTypeMatcher)
		return ok && matcher.MatchesArtifactType(artifactType)
	})
}
