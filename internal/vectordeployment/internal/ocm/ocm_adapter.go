package ocm

//go:generate go run go.uber.org/mock/mockgen -destination=mocks/mock_ocm_client.go -package=mocks github.com/konfidence-project/konfidence/pkg/ocm/repository Client

import (
	"context"
	"encoding/json"
	"fmt"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	pkgocm "github.com/konfidence-project/konfidence/pkg/ocm/repository"
	"github.com/konfidence-project/konfidence/pkg/ocm/util"
	"ocm.software/open-component-model/bindings/go/credentials"
	descruntime "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	ocmruntime "ocm.software/open-component-model/bindings/go/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/konfidence-project/konfidence/internal/vectordeployment/internal/controller"
)

// Adapter implements the VectorOcmPort interface.
type Adapter struct {
	client                     pkgocm.Client
	verifier                   crypto.Verifier
	resolver                   credentials.Resolver
	vectorSpecs, artifactSpecs []crypto.SignatureSpec
}

var _ controller.VectorOcmPort = (*Adapter)(nil)

// NewAdapter builds an Adapter from a shared Verifier, a resolver, and two
// spec slices. An empty spec slice disables verification for that code path
// (the shared Verifier no-ops on empty specs).
func NewAdapter(
	ctx context.Context,
	resolver credentials.Resolver,
	verifier crypto.Verifier,
	vectorSpecs, artifactSpecs []crypto.SignatureSpec,
) (Adapter, error) {
	ocmClient, err := pkgocm.NewOciClientBuilder().
		WithLogger(ctrl.Log).
		WithResolver(resolver).
		Build(ctx)

	if err != nil {
		return Adapter{}, fmt.Errorf("unable to build ocm client: %w", err)
	}

	return Adapter{
		client:        ocmClient,
		verifier:      verifier,
		resolver:      resolver,
		vectorSpecs:   vectorSpecs,
		artifactSpecs: artifactSpecs,
	}, nil
}

// NewAdapterWithClient creates an Adapter using the provided client. Intended for testing.
// Verification is disabled: the verifier is built with no specs, and the empty
// vector/artifact spec slices make every Verify call a no-op.
func NewAdapterWithClient(client pkgocm.Client) Adapter {
	verifier, _ := crypto.NewVerifierBuilder().Build()
	return Adapter{client: client, verifier: verifier}
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

	if err := a.verifier.Verify(ctx, a.resolver, a.vectorSpecs, []*descruntime.Descriptor{&descriptor}); err != nil {
		return controller.VectorDescriptor{}, permanent(konfidence.VectorDeploymentStalledReasonSignatureVerificationFailed,
			fmt.Errorf("unable to verify ocm descriptor for reference (%s): %w", ref.String(), err))
	}

	// Extract the optional, singleton vector-scoped configuration resource and strip it from the descriptor before V2
	// serialization. The resource is a Konfidence-internal concern, not part of the public OCM contract that is later
	// re-read from the persisted JSON, so removing it keeps the V2 representation clean and avoids serialization
	// problems for access types that may not be wired into the runtime registry of every consumer (e.g. tests).
	vectorConfig, prunedResources, err := util.ExtractVectorConfigResource(ctx, a.client, ref, descriptor.Component.Resources)
	if err != nil {
		return controller.VectorDescriptor{}, err
	}
	descriptor.Component.Resources = prunedResources

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
		Configuration:  vectorConfig,
	}, nil
}

// GetArtifactManifestByReference fetches the artifact manifest for the given artifact reference.
func (a Adapter) GetArtifactManifestByReference(ctx context.Context, ref compref.Ref) (controller.ArtifactManifest, error) {
	descriptor, err := a.client.Get(ctx, ref)
	if err != nil {
		return controller.ArtifactManifest{}, fmt.Errorf("failed to fetch artifact descriptor for %s: %w", ref.String(), err)
	}

	if err := a.verifier.Verify(ctx, a.resolver, a.artifactSpecs, []*descruntime.Descriptor{&descriptor}); err != nil {
		return controller.ArtifactManifest{}, permanent(konfidence.VectorDeploymentStalledReasonSignatureVerificationFailed,
			fmt.Errorf("unable to verify ocm descriptor for reference (%s): %w", ref.String(), err))
	}

	var konfidenceArtifactManifest *controller.ArtifactManifest
	var taskManifests []controller.TaskManifest
	artifactResources := make([]controller.OCMResource, 0, len(descriptor.Component.Resources))

	for i := range descriptor.Component.Resources {
		resource := &descriptor.Component.Resources[i]

		switch resource.Type {
		case konfidenceResourceTypeArtifactManifest:
			data, err := util.GetLocalResourceBlob(ctx, a.client, ref, resource)
			if err != nil {
				return controller.ArtifactManifest{}, fmt.Errorf("failed to get manifest blob for resource %s in component %s: %w", resource.Name, ref.String(), err)
			}

			var konfidenceManifestDto struct {
				Type       konfidence.ArtifactType `json:"type"`
				AllowReuse bool                    `json:"allowReuse"`
			}
			if err := json.Unmarshal(data, &konfidenceManifestDto); err != nil {
				return controller.ArtifactManifest{}, permanent(konfidence.VectorDeploymentStalledReasonManifestInvalid,
					fmt.Errorf("failed to unmarshal manifest data for resource %s in component %s: %w", resource.Name, ref.String(), err))
			}

			konfidenceArtifactManifest = &controller.ArtifactManifest{
				Type:       konfidenceManifestDto.Type,
				AllowReuse: konfidenceManifestDto.AllowReuse,
			}

		case konfidenceResourceTypeArtifactTaskManifest:
			data, err := util.GetLocalResourceBlob(ctx, a.client, ref, resource)
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
				return controller.ArtifactManifest{}, permanent(konfidence.VectorDeploymentStalledReasonManifestInvalid,
					fmt.Errorf("failed to unmarshal task manifest data for resource %s in component %s: %w", resource.Name, ref.String(), err))
			}

			taskManifests = append(taskManifests, controller.TaskManifest{
				Name:      konfidenceTaskManifestDto.Name,
				Type:      konfidenceTaskManifestDto.Type,
				DependsOn: konfidenceTaskManifestDto.DependsOn,
				Spec:      string(konfidenceTaskManifestDto.Spec),
			})

		default:
			if resource.Access == nil {
				return controller.ArtifactManifest{}, permanent(konfidence.VectorDeploymentStalledReasonManifestInvalid,
					fmt.Errorf("missing access spec for resource %s in component %s", resource.Name, ref.String()))
			}
			data, err := json.Marshal(resource.Access)
			if err != nil {
				return controller.ArtifactManifest{}, permanent(konfidence.VectorDeploymentStalledReasonManifestInvalid,
					fmt.Errorf("failed to marshal access spec for resource %s in component %s: %w", resource.Name, ref.String(), err))
			}

			artifactResources = append(artifactResources, controller.OCMResource{
				Name:    resource.Name,
				Type:    resource.Type,
				Content: data,
			})
		}
	}

	if konfidenceArtifactManifest == nil {
		return controller.ArtifactManifest{}, permanent(konfidence.VectorDeploymentStalledReasonManifestMissing,
			fmt.Errorf("no artifact manifest found in component %s", ref.String()))
	}

	konfidenceArtifactManifest.Tasks = taskManifests
	konfidenceArtifactManifest.Resources = artifactResources

	return *konfidenceArtifactManifest, nil
}

func permanent(reason string, err error) error {
	return &controller.PermanentError{Reason: reason, Err: err}
}
