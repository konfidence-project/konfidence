package v1alpha1

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
)

// ArtifactType is a type-and-domain identifier for a deployer's artifactType contract. Its syntax is defined in
// section 1.1 of conformance/specs/deployer.md. It is encouraged to use MustArtifactType for construction.
type ArtifactType string

// ArtifactTypeProvider is implemented by API resources that declare an artifact type.
type ArtifactTypeProvider interface {
	ArtifactType() ArtifactType
}

// artifactTypeRegex implements the <artifact-type> grammar in
// conformance/specs/deployer.md section 1.1.
var artifactTypeRegex = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`)

// NewArtifactType validates s against the artifact type grammar and returns it
// as an ArtifactType.
func NewArtifactType(s string) (ArtifactType, error) {
	if !artifactTypeRegex.MatchString(s) {
		return "", fmt.Errorf("invalid artifact type: %q", s)
	}

	return ArtifactType(s), nil
}

// MustArtifactType returns s as an ArtifactType, panicking if it is invalid.
func MustArtifactType(s string) ArtifactType {
	t, err := NewArtifactType(s)
	if err != nil {
		panic(err)
	}
	return t
}

// Valid reports whether the ArtifactType matches the artifact type grammar.
func (t ArtifactType) Valid() bool {
	return artifactTypeRegex.MatchString(string(t))
}

// ArtifactTypeSet is a set of ArtifactTypes.
type ArtifactTypeSet map[ArtifactType]struct{}

// String returns the artifact types in t in sorted order.
func (t ArtifactTypeSet) String() string {
	return fmt.Sprintf("%v", slices.Sorted(maps.Keys(t)))
}

// Contains reports whether the set contains the given ArtifactType.
func (t ArtifactTypeSet) Contains(artifactType ArtifactType) bool {
	_, ok := t[artifactType]
	return ok
}
