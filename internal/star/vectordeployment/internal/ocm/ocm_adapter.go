package ocm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	pkgocm "github.com/konfidence-project/konfidence/pkg/ocm/repository"
	corev1 "k8s.io/api/core/v1"
	descruntime "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	ocmruntime "ocm.software/open-component-model/bindings/go/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/konfidence-project/konfidence/internal/star/vectordeployment/internal/controller"
)

// Adapter implements the VectorOcmPort interface.
type Adapter struct {
	client                           pkgocm.Client
	vectorVerifier, artifactVerifier crypto.Verifier
}

var _ controller.VectorOcmPort = (*Adapter)(nil)

func NewAdapter(ctx context.Context, secret *corev1.Secret, vectorVerifier, artifactVerifier crypto.Verifier) (Adapter, error) {
	//nolint:staticcheck
	ocmClient, err := pkgocm.NewOciClientBuilder().
		WithLogger(ctrl.Log).
		WithDockerConfigJsonSecret(secret).
		Build(ctx)

	if err != nil {
		return Adapter{}, fmt.Errorf("unable to build ocm client: %w", err)
	}

	return Adapter{client: ocmClient, vectorVerifier: vectorVerifier, artifactVerifier: artifactVerifier}, nil
}

// NewAdapterWithClient creates an Adapter using the provided client. Intended for testing.
func NewAdapterWithClient(client pkgocm.Client) Adapter {
	return Adapter{client: client, vectorVerifier: crypto.NoopVerifier{}, artifactVerifier: crypto.NoopVerifier{}}
}

const (
	konfidenceResourceTypeArtifactManifest     = "cloud.konfidence.artifact.manifest"
	konfidenceResourceTypeArtifactTaskManifest = "cloud.konfidence.artifact.task.manifest"
)

func (a Adapter) GetVectorDescriptor(ctx context.Context, ref compref.Ref) (controller.VectorDescriptor, error) {
	descriptor, err := a.client.Get(ctx, ref)
	if err != nil {
		return controller.VectorDescriptor{},
			fmt.Errorf("unable to get descriptor for component %q version %q: %w", ref.Component, ref.Version, err)
	}

	if err := a.vectorVerifier.Verify(ctx, &descriptor); err != nil {
		return controller.VectorDescriptor{}, fmt.Errorf("unable to verify ocm descriptor for reference (%s): %w",
			ref.String(), err)
	}

	// Convert runtime descriptor to v2 and then marshal to JSON.
	// descruntime.Descriptor.MarshalJSON() is intentionally unsupported; ConvertToV2 is required.
	v2Desc, err := descruntime.ConvertToV2(ocmruntime.NewScheme(), &descriptor)
	if err != nil {
		return controller.VectorDescriptor{},
			fmt.Errorf("failed to convert descriptor to v2 for component %q version %q: %w",
				descriptor.Component.Name, descriptor.Component.Version, err)
	}
	v2DescriptorJSON, err := json.Marshal(v2Desc)
	if err != nil {
		return controller.VectorDescriptor{},
			fmt.Errorf("failed to marshal component spec for component %q version %q: %w",
				descriptor.Component.Name, descriptor.Component.Version, err)
	}

	refs := make([]compref.Ref, len(descriptor.Component.References))
	for i, componentRef := range descriptor.Component.References {
		refs[i] = compref.Ref{
			Repository: ref.Repository,
			Component:  componentRef.Component,
			Version:    componentRef.Version,
		}
	}

	return controller.VectorDescriptor{
		References:     refs,
		DescriptorJSON: v2DescriptorJSON,
	}, nil
}

// GetArtifactManifestByReference fetches the artifact manifest for the given artifact reference.
func (a Adapter) GetArtifactManifestByReference(ctx context.Context, ref compref.Ref) (controller.ArtifactManifest, error) {
	descriptor, err := a.client.Get(ctx, ref)
	if err != nil {
		return controller.ArtifactManifest{}, fmt.Errorf("failed to fetch artifact descriptor for %s: %w", ref.String(), err)
	}

	if err := a.artifactVerifier.Verify(ctx, &descriptor); err != nil {
		return controller.ArtifactManifest{}, fmt.Errorf("unable to verify ocm descriptor for reference (%s): %w",
			ref.String(), err)
	}

	var konfidenceArtifactManifest *controller.ArtifactManifest
	var taskManifests []controller.TaskManifest
	artifactResources := make([]controller.OCMResource, 0, len(descriptor.Component.Resources))

	for i := range descriptor.Component.Resources {
		resource := &descriptor.Component.Resources[i]

		switch resource.Type {
		case konfidenceResourceTypeArtifactManifest:
			data, err := a.getLocalResourceBlob(ctx, ref, resource)
			if err != nil {
				return controller.ArtifactManifest{}, fmt.Errorf("failed to get manifest blob for resource %s in component %s: %w", resource.Name, ref.String(), err)
			}

			var konfidenceManifestDto struct {
				Type       string `json:"type"`
				AllowReuse bool   `json:"allowReuse"`
			}
			if err := json.Unmarshal(data, &konfidenceManifestDto); err != nil {
				return controller.ArtifactManifest{}, fmt.Errorf("failed to unmarshal manifest data for resource %s in component %s: %w", resource.Name, ref.String(), err)
			}

			konfidenceArtifactManifest = &controller.ArtifactManifest{
				Type:       konfidenceManifestDto.Type,
				AllowReuse: konfidenceManifestDto.AllowReuse,
			}

		case konfidenceResourceTypeArtifactTaskManifest:
			data, err := a.getLocalResourceBlob(ctx, ref, resource)
			if err != nil {
				return controller.ArtifactManifest{}, fmt.Errorf("failed to get task manifest blob for resource %s in component %s: %w", resource.Name, ref.String(), err)
			}

			var konfidenceTaskManifestDto struct {
				Name      string          `json:"name"`
				Type      string          `json:"type"`
				DependsOn []string        `json:"dependsOn"`
				Spec      json.RawMessage `json:"spec"`
			}
			if err := json.Unmarshal(data, &konfidenceTaskManifestDto); err != nil {
				return controller.ArtifactManifest{},
					fmt.Errorf("failed to unmarshal task manifest data for resource %s in component %s: %w", resource.Name, ref.String(), err)
			}

			taskManifests = append(taskManifests, controller.TaskManifest{
				Name:      konfidenceTaskManifestDto.Name,
				Type:      konfidenceTaskManifestDto.Type,
				DependsOn: konfidenceTaskManifestDto.DependsOn,
				Spec:      string(konfidenceTaskManifestDto.Spec),
			})

		default:
			if resource.Access == nil {
				return controller.ArtifactManifest{}, fmt.Errorf("missing access spec for resource %s in component %s", resource.Name, ref.String())
			}
			data, err := json.Marshal(resource.Access)
			if err != nil {
				return controller.ArtifactManifest{}, fmt.Errorf("failed to marshal access spec for resource %s in component %s: %w", resource.Name, ref.String(), err)
			}

			artifactResources = append(artifactResources, controller.OCMResource{
				Name:    resource.Name,
				Type:    resource.Type,
				Content: data,
			})
		}
	}

	if konfidenceArtifactManifest == nil {
		return controller.ArtifactManifest{}, fmt.Errorf("no artifact manifest found in component %s", ref.String())
	}

	konfidenceArtifactManifest.Tasks = taskManifests
	konfidenceArtifactManifest.Resources = artifactResources

	return *konfidenceArtifactManifest, nil
}

// getLocalResourceBlob retrieves the raw bytes of a locally-stored resource blob.
func (a Adapter) getLocalResourceBlob(ctx context.Context, ref compref.Ref, resource *descruntime.Resource) ([]byte, error) {
	identity := resource.ToIdentity()
	b, _, err := a.client.GetLocalResource(ctx, ref, identity)
	if err != nil {
		return nil, fmt.Errorf("getting local resource for %s identity %v: %w", ref, identity, err)
	}
	rc, err := b.ReadCloser()
	if err != nil {
		return nil, fmt.Errorf("opening blob reader: %w", err)
	}
	defer func() { _ = rc.Close() }()
	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("reading blob content: %w", err)
	}
	return data, nil
}
