package vector

import (
	pkgocm "github.com/konfidence-project/konfidence/pkg/ocm/repository"
	"ocm.software/open-component-model/bindings/go/runtime"
)

var (
	ErrVectorNotFound = pkgocm.ErrNotFound
)

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
