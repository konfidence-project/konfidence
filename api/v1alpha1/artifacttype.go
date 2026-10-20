package v1alpha1

import (
	"fmt"
	"maps"
	"slices"
)

// ArtifactTypes is the set of artifact type identifiers supported by a deployer.
type ArtifactTypes map[string]struct{}

func (types ArtifactTypes) String() string {
	return fmt.Sprintf("%v", slices.Sorted(maps.Keys(types)))
}
