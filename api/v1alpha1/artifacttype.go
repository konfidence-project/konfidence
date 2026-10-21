package v1alpha1

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
)

type ArtifactType string

// artifactTypeRegex implements the <artifact-type> grammar in
// conformance/specs/deployer.md section 1.1.
var artifactTypeRegex = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`)

func NewArtifactType(s string) (ArtifactType, error) {
	if !artifactTypeRegex.MatchString(s) {
		return "", fmt.Errorf("invalid artifact type: %q", s)
	}

	return ArtifactType(s), nil
}

func MustArtifactType(s string) ArtifactType {
	t, err := NewArtifactType(s)
	if err != nil {
		panic(err)
	}
	return t
}

func (t ArtifactType) Valid() bool {
	return !artifactTypeRegex.MatchString(string(t))
}

type ArtifactTypeSet map[ArtifactType]struct{}

func (t ArtifactTypeSet) String() string {
	return fmt.Sprintf("%v", slices.Sorted(maps.Keys(t)))
}

func (t ArtifactTypeSet) Contains(artifactType ArtifactType) bool {
	_, ok := t[artifactType]
	return ok
}
