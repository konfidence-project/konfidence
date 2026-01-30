package domain

//go:generate go run go.uber.org/mock/mockgen -source=vector.go -destination=mocks/mock_ocm_port.go -package=mocks

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"ocm.software/open-component-model/bindings/go/runtime"
)

var ErrVectorNotFound = fmt.Errorf("vector not found in OCM repository")

// VectorOcmPort defines the interface for interacting with the OCM repository for vector operations.
type VectorOcmPort interface {

	// GetLatestArtifactVersions resolves the versions of the given components in the OCM repository.
	GetLatestArtifactVersions(ctx context.Context, references []OcmReference) ([]Artifact, error)

	// GetLatestVector retrieves the latest vector from the OCM repository.
	GetLatestVector(ctx context.Context, vectorRef OcmReference) (Vector, error)

	// CreateVector creates a new vector in the OCM repository.
	CreateVector(ctx context.Context, vector Vector) error
}

type Vector struct {
	Version   string
	Reference OcmReference
	Artifacts []Artifact
}

type Artifact struct {
	Version      string
	OcmReference OcmReference
}

type OcmReference struct {
	Component  string
	Repository string // todo: will removed in future, when repository is configurable
}

func (o OcmReference) String() string {
	return fmt.Sprintf("%s//%s", o.Repository, o.Component)
}

func NewOcmReference(vectorTemplateComponentName string) (OcmReference, error) {
	component, err := extractOCMComponent(vectorTemplateComponentName)
	if err != nil {
		return OcmReference{}, fmt.Errorf("unable to extract ocm component from vector template component name (%s): %w",
			vectorTemplateComponentName, err)
	}
	repositoryUrl, err := extractRepositoryURL(vectorTemplateComponentName)
	if err != nil {
		return OcmReference{}, fmt.Errorf("unable to extract registry url from vector template component name (%s): %w",
			vectorTemplateComponentName, err)
	}

	return OcmReference{
		Component:  component,
		Repository: repositoryUrl.Host + repositoryUrl.Path,
	}, nil
}

// Can be removed once we remove the registry from the Vector Template components
func extractOCMComponent(registryAndOCMComponent string) (string, error) {
	registryUrl, err := extractRepositoryURL(registryAndOCMComponent)
	if err != nil {
		return "", fmt.Errorf("unable to cut registry url from input (%s) to extract ocm component. Failed to extract registry url: %w",
			registryAndOCMComponent, err)
	}
	registry := strings.TrimPrefix(registryUrl.String(), "//")
	ocmComponent := strings.TrimPrefix(registryAndOCMComponent, registry+"//")
	return ocmComponent, nil
}

// Can be removed once we remove the registry from the Vector Template components
func extractRepositoryURL(registryAndOCMComponent string) (*url.URL, error) {
	data := strings.Split(registryAndOCMComponent, "//")
	var (
		err         error
		registryUrl *url.URL
	)
	switch len(data) {
	case 3:
		registryStr := data[0] + "//" + data[1]
		if registryUrl, err = url.Parse(registryStr); err != nil {
			return nil, fmt.Errorf("unable parse *url.URL from identified oci url.URL{}: %s: %w", registryStr, err)
		}
	case 2:
		if registryUrl, err = runtime.ParseURLAndAllowNoScheme(data[0]); err != nil {
			return nil, fmt.Errorf("unable parse *url.URL from identified oci url.URL{}: %s: %w", data[0], err)
		}
	default:
		return nil, fmt.Errorf("unable to parse oci url.URL{} from registryAndOCMComponent input: %s. Format is not supported", registryAndOCMComponent)
	}
	return registryUrl, nil
}

func HasDrift(desired, actual []Artifact) bool {
	if len(desired) != len(actual) {
		return true
	}

	for _, desiredElement := range desired {
		desiredElementFound := false
		for _, actualElement := range actual {
			if desiredElement.OcmReference.Component == actualElement.OcmReference.Component {
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
