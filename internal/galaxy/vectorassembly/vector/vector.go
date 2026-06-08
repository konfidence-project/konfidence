package vector

import (
	"errors"

	"ocm.software/open-component-model/bindings/go/runtime"
)

// ErrVectorNotFound indicates that a requested vector could not be found in the OCM repository.
var ErrVectorNotFound = errors.New("vector not found")

type Vector struct {
	Version   string
	Name      string
	Artifacts []Artifact
}

type Artifact struct {
	Version    string
	Name       string
	Digest     string
	SourceRepo runtime.Typed
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
