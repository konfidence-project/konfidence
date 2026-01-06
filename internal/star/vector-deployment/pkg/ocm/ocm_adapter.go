package ocm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	utilErrors "github.com/mandelsoft/goutils/errors"
	"ocm.software/ocm/api/oci"
	"ocm.software/ocm/api/ocm"
	"ocm.software/ocm/api/ocm/extensions/repositories/ocireg"

	"github.com/konfidence-project/landscape-vector-deployment-controller/internal/controller/domain"
)

// Adapter implements the VectorOcmPort interface.
type Adapter struct {
	provider ContextProvider
}

var _ domain.VectorOcmPort = (*Adapter)(nil)

func NewOcmAdapter(provider ContextProvider) Adapter {
	return Adapter{
		provider: provider,
	}
}

func (a Adapter) GetArtifactManifestByReference(ctx context.Context, namespace string,
	ociUrl string, artifactName domain.ArtifactReference) (*domain.ArtifactManifest, error) {
	ocmRef, err := parseComponentVersionUrl(ociUrl)
	if err != nil {
		return nil, err
	}
	ocmRef.Component = artifactName.ComponentName
	ocmRef.Version = &artifactName.Version

	ocmCtx, err := a.provider.GetOCMContext(ctx, namespace, ociUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to get OCM context, %w", err)
	}

	// fetch component version access from OCM
	componentVersionAccess, err := fetchComponentVersionAccess(ocmCtx, *ocmRef)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch artifact OCM for reference %q: %w", ocmRef, err)
	}

	// get artifactManifest and the associated task manifests from component version access
	var artifactManifest *domain.ArtifactManifest
	var taskManifests []domain.TaskManifest
	artifactResources := make([]domain.OCMResource, 0, len(componentVersionAccess.GetResources()))

	for _, resource := range componentVersionAccess.GetResources() {
		if resource.Meta().Type == "cloud.konfidence.artifact.manifest" {
			accessMethod, err := resource.AccessMethod()
			if err != nil {
				return nil, fmt.Errorf("failed to access method for resource %s in component %s: %w", resource.Meta().Name, ocmRef, err)
			}
			data, err := accessMethod.Get()
			if err != nil {
				return nil, fmt.Errorf("failed to get manifest data for resource %s in component %s: %w", resource.Meta().Name, ocmRef, err)
			}

			// map raw JSON data to ArtifactManifest struct
			var artifactManifestDto struct {
				Type       string `json:"type"`
				AllowReuse bool   `json:"allowReuse"`
			}
			err = json.Unmarshal(data, &artifactManifestDto)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal manifest data for resource %s in component %s: %w", resource.Meta().Name, ocmRef, err)
			}

			artifactManifest = &domain.ArtifactManifest{
				Name:       ocmRef.Component,
				Version:    *ocmRef.Version,
				Type:       artifactManifestDto.Type,
				AllowReuse: artifactManifestDto.AllowReuse,
				Tasks:      nil,
			}

			continue
		}
		if resource.Meta().Type == "cloud.konfidence.artifact.task.manifest" {
			accessMethod, _ := resource.AccessMethod()
			data, _ := accessMethod.Get()
			var taskManifestDto struct {
				Name      string          `json:"name"`
				Type      string          `json:"type"`
				DependsOn []string        `json:"dependsOn"`
				Spec      json.RawMessage `json:"spec"`
			}
			err = json.Unmarshal(data, &taskManifestDto)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal task manifest data for resource %s in component %s: %w", resource.Meta().Name, ocmRef, err)
			}

			// map to domain.TaskManifest
			taskManifests = append(taskManifests, domain.TaskManifest{
				Name:      taskManifestDto.Name,
				Type:      taskManifestDto.Type,
				DependsOn: taskManifestDto.DependsOn,
				Spec:      string(taskManifestDto.Spec),
			})

			continue
		}

		resourceAccess, err := resource.Access()
		if err != nil {
			return nil, fmt.Errorf("failed to get access for resource %s in component %s: %w", resource.Meta().Name, artifactName.ComponentName, err)
		}

		genericAccessSpec, err := ocmCtx.AccessSpecForSpec(resourceAccess)
		if err != nil {
			return nil, fmt.Errorf("failed to get effective access spec for resource %s in component %s: %w", resource.Meta().Name, artifactName.ComponentName, err)
		}

		var buf bytes.Buffer
		err = json.NewEncoder(&buf).Encode(genericAccessSpec)
		if err != nil {
			return nil, fmt.Errorf("failed to encode generic access spec for resource %s in component %s: %w", resource.Meta().Name, artifactName.ComponentName, err)
		}

		artifactResources = append(artifactResources, domain.OCMResource{
			Name:    resource.Meta().Name,
			Content: buf.Bytes(),
			Type:    resource.Meta().Type,
		})
	}

	if artifactManifest == nil {
		return nil, fmt.Errorf("no artifact manifest found in component %s", ocmRef)
	}

	artifactManifest.Tasks = taskManifests
	artifactManifest.Resources = artifactResources

	return artifactManifest, nil
}

func (a Adapter) GetVectorByReference(ctx context.Context, namespace string, vectorReference domain.VectorReference) (*domain.Vector, error) {
	// 1. map vectorRef to ocm.RefSpec
	ocmRef, err := parseComponentVersionUrl(vectorReference.OciRegistryUrl)
	if err != nil {
		return nil, err
	}

	ocmCtx, err := a.provider.GetOCMContext(ctx, namespace, vectorReference.OciRegistryUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to get OCM context %w", err)
	}

	// 2. fetch component version access from OCM
	componentVersionAccess, err := fetchComponentVersionAccess(ocmCtx, *ocmRef)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch vector OCM for reference %q: %w", vectorReference, err)
	}

	descriptor := componentVersionAccess.GetDescriptor()
	if descriptor == nil {
		return nil, fmt.Errorf("no component descriptor for component %q version %q", ocmRef.Component, *ocmRef.Version)
	}

	// 4. marshal component spec to JSON string
	componentSpec, err := json.Marshal(descriptor.ComponentSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal component spec for component %q version %q: %w", ocmRef.Component, *ocmRef.Version, err)
	}

	// 5. map to domain.Vector
	vector := domain.Vector{
		Reference:     vectorReference,
		ComponentSpec: string(componentSpec),
		Artifacts:     nil,
	}

	vector.Artifacts = make([]domain.ArtifactReference, len(descriptor.GetReferences()))
	for i, componentRef := range descriptor.GetReferences() {
		vector.Artifacts[i] = domain.ArtifactReference{
			Version:       componentRef.Version,
			ComponentName: componentRef.ComponentName,
		}
	}
	return &vector, nil
}

func parseComponentVersionUrl(ref string) (*ocm.RefSpec, error) {
	ocmRef, err := ocm.ParseRef(ref)
	if err != nil {
		return nil, fmt.Errorf("invalid reference %q: %w", ref, err)
	}
	return &ocmRef, nil
}

// connectToOciRepository connects to an OCI repository based on the provided reference.
// The Caller is responsible for closing the repository!  Call `defer repo.Close()` after a successful call.
func connectToOciRepository(ctx ocm.Context, ref ocm.RefSpec) (ocm.Repository, error) {
	consumerId, err := oci.GetConsumerIdForRef(ref.Host)
	if err != nil {
		return nil, fmt.Errorf("failed to get consumer ID for OCI host %q: %w", ref.Host, err)
	}
	credentials, err := ctx.CredentialsContext().GetCredentialsForConsumer(consumerId)
	if err != nil && !utilErrors.IsErrUnknownKind(err, "consumer") {
		return nil, fmt.Errorf("failed to get credentials for consumer %q: %w", consumerId.String(), err)
	}

	spec := ocireg.NewRepositorySpec(ref.UniformRepositorySpec.String())

	repo, err := ctx.RepositoryForSpec(spec, credentials)
	if err != nil {
		return nil, fmt.Errorf("cannot setup repository: %w", err)
	}
	return repo, nil
}

func fetchComponentVersionAccess(ctx ocm.Context, ref ocm.RefSpec) (ocm.ComponentVersionAccess, error) {
	// 1. connect to OCI repository
	repo, err := connectToOciRepository(ctx, ref)
	if err != nil {
		return nil, err
	}
	defer func() { _ = repo.Close() }()

	// 2. fetch component version access from repository
	componentVersionAccess, err := repo.LookupComponentVersion(ref.Component, *ref.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch component version %q from repository %q: %w", ref.Component, ref.UniformRepositorySpec.String(), err)
	}

	return componentVersionAccess, nil
}
