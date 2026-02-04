package ocm

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"

	"github.com/konfidence-project/gcp-vector-assembly-controller/internal/controller/domain"
	ocmDescriptor "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	"ocm.software/open-component-model/bindings/go/oci"
	urlresolver "ocm.software/open-component-model/bindings/go/oci/resolver/url"
	"ocm.software/open-component-model/bindings/go/repository"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	_                    domain.VectorOcmPort = (*Adapter)(nil)
	ErrComponentNotFound                      = errors.New("component not found in OCM repository")
)

// Adapter is an implementation of the VectorOcmPort interface that interacts with OCM repositories to manage vectors and their associated artifacts.
type Adapter struct {
	vectorVerifier, artifactVerifier Verifier
	vectorSigner                     Signer
	digester                         Digester
}

func (a Adapter) GetLatestArtifactVersions(ctx context.Context, references []domain.OcmReference) ([]domain.Artifact, error) {
	if len(references) == 0 {
		return nil, nil
	}
	artifacts, unverifiedDescriptors := make([]domain.Artifact, 0, len(references)), make([]*ocmDescriptor.Descriptor, 0, len(references))
	for _, ref := range references {
		version, err := a.getLatestComponentVersion(ctx, ref)
		if err != nil {
			return nil, fmt.Errorf("unable to get latest version for ocm component (%s): %w", ref, err)
		}
		desc, err := a.getDescriptorForVersion(ctx, ref, version)
		if err != nil {
			return nil,
				fmt.Errorf("unable to get descriptor for latest ocm component version (%s) for component (%s): %w", version, ref, err)
		}
		unverifiedDescriptors = append(unverifiedDescriptors, desc)
		dig, err := a.digester.GenerateDigest(ctx, desc)
		if err != nil {
			return nil, fmt.Errorf("unable to generate digest for ocm descriptor for component (%s) version (%s): %w", ref, version, err)
		}
		artifact := domain.Artifact{
			OcmReference: ref,
			Version:      version,
			Digest:       dig.Value,
		}
		artifacts = append(artifacts, artifact)
	}
	if err := a.artifactVerifier.Verify(ctx, unverifiedDescriptors...); err != nil {
		return nil, fmt.Errorf("ocm artifact verification failed for one or more artifacts: %w", err)
	}
	return artifacts, nil
}

func (a Adapter) GetLatestVector(ctx context.Context, vectorRef domain.OcmReference) (domain.Vector, error) {
	version, err := a.getLatestComponentVersion(ctx, vectorRef)
	if errors.Is(err, ErrComponentNotFound) {
		return domain.Vector{}, errors.Join(domain.ErrVectorNotFound, err)
	}
	if err != nil {
		return domain.Vector{}, fmt.Errorf("unable to get latest version for vector ocm component (%s): %w", vectorRef, err)
	}
	descriptor, err := a.getDescriptorForVersion(ctx, vectorRef, version)
	if err != nil {
		return domain.Vector{}, fmt.Errorf("unable to get descriptor for latest ocm component version (%s) for vector (%s): %w", version, vectorRef, err)
	}
	if err := a.vectorVerifier.Verify(ctx, descriptor); err != nil {
		return domain.Vector{}, fmt.Errorf("unable to verify ocm descriptor for vector (%s) version (%s): %w",
			vectorRef, version, err)
	}
	return mapToDomain(version, vectorRef, descriptor.Component.References), nil
}

func (a Adapter) CreateVector(ctx context.Context, vector domain.Vector) error {
	vectorDescriptor := mapToDescriptor(vector, a.digester.GetHashAlgorithm().String(), a.digester.GetNormalisationAlgorithm())
	if err := a.vectorSigner.Sign(ctx, &vectorDescriptor); err != nil {
		return fmt.Errorf("unable to Sign ocm descriptor for vector (%s): %w", vector.Reference, err)
	}
	if err := a.addOcmComponent(ctx, vector.Reference, &vectorDescriptor); err != nil {
		return fmt.Errorf("unable to add ocm component for vector (%s): %w", vector.Reference, err)
	}
	return nil
}

func (a Adapter) addOcmComponent(ctx context.Context, reference domain.OcmReference, desc *ocmDescriptor.Descriptor) error {
	repo, err := a.getOcmComponentVersionRepository(reference)
	if err != nil {
		return fmt.Errorf("unable to get ocm component version repository for %s: %w", reference, err)
	}

	// safety check - if the component version already exists, we return an error
	_, err = repo.GetComponentVersion(ctx, desc.Component.Name, desc.Component.Version)
	if errors.Is(err, repository.ErrNotFound) {
		if err := repo.AddComponentVersion(ctx, desc); err != nil {
			return fmt.Errorf("unable to add component version to ocm repository for %s: %w", reference, err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("unable to check existence of component version in ocm repository for %s: %w", reference, err)
	}

	return fmt.Errorf("component version %s:%s already exists in ocm repository for %s", desc.Component.Name, desc.Component.Version, reference)
}

func (a Adapter) getLatestComponentVersion(ctx context.Context, reference domain.OcmReference) (string, error) {
	repo, err := a.getOcmComponentVersionRepository(reference)
	if err != nil {
		return "", fmt.Errorf("unable to get ocm component version repository for %s: %w", reference, err)
	}

	componentVersions, err := repo.ListComponentVersions(ctx, reference.Component)
	if err != nil && strings.Contains(err.Error(), "repository name not known to registry") {
		return "", fmt.Errorf("no versions found for component %s: %w", reference.Component, ErrComponentNotFound)
	} else if err != nil {
		return "", fmt.Errorf("unable to list component versions for %s: %w", reference.Component, err)
	}

	if len(componentVersions) == 0 {
		return "", fmt.Errorf("no versions found for component %s: %w", reference.Component, ErrComponentNotFound)
	}
	return componentVersions[0], nil
}

func (a Adapter) getDescriptorForVersion(ctx context.Context, reference domain.OcmReference, version string) (*ocmDescriptor.Descriptor, error) {
	repo, err := a.getOcmComponentVersionRepository(reference)
	if err != nil {
		return nil, fmt.Errorf("unable to get ocm component version repository for %s: %w",
			reference.Repository, err)
	}
	componentVersionDescriptor, err := repo.GetComponentVersion(ctx, reference.Component, version)
	if err != nil {
		return nil, fmt.Errorf("unable to get ocm component version descriptor for component %s version %s: %w",
			reference.Component, version, err)
	}
	return componentVersionDescriptor, nil
}

func (a Adapter) getOcmComponentVersionRepository(reference domain.OcmReference) (repository.ComponentVersionRepository, error) {
	// todo: quick and dirty auth client for local testing only!!
	authClient := &auth.Client{
		Client: retry.DefaultClient,
		Header: map[string][]string{"User-Agent": {"gcp-vector-assembly-controller"}},
		Credential: auth.StaticCredential(reference.Repository, auth.Credential{
			Username: "",
			Password: "",
		}),
	}
	resolver, err := urlresolver.New(
		urlresolver.WithBaseURL(reference.Repository),
		urlresolver.WithPlainHTTP(true),
		urlresolver.WithBaseClient(authClient),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create url resolver for repository: %s: %w", reference.Repository, err)
	}
	repo, err := oci.NewRepository(oci.WithResolver(resolver))
	if err != nil {
		return nil, fmt.Errorf("unable to create oci repository for repository: %s: %w", reference.Repository, err)
	}
	return repo, nil
}

func mapToDomain(version string, vectorRef domain.OcmReference, artifactRefs []ocmDescriptor.Reference) domain.Vector {
	artifacts := make([]domain.Artifact, 0, len(artifactRefs))
	for _, ref := range artifactRefs {
		artifact := domain.Artifact{
			Version: ref.Version,
			OcmReference: domain.OcmReference{
				Component:  ref.Component,
				Repository: "", // will be removed when repository is configurable
			},
			Digest: ref.Digest.Value,
		}
		artifacts = append(artifacts, artifact)
	}
	return domain.Vector{
		Version:   version,
		Reference: vectorRef,
		Artifacts: artifacts,
	}
}

func mapToDescriptor(vector domain.Vector, hashAlgo, normAlgo string) ocmDescriptor.Descriptor {
	latestArtifacts := make([]ocmDescriptor.Reference, 0, len(vector.Artifacts))
	for _, artifact := range vector.Artifacts {
		latestArtifacts = append(latestArtifacts, mapToReference(artifact, hashAlgo, normAlgo))
	}
	return ocmDescriptor.Descriptor{
		Meta: ocmDescriptor.Meta{
			Version: "v2",
		},
		Component: ocmDescriptor.Component{
			ComponentMeta: ocmDescriptor.ComponentMeta{
				ObjectMeta: ocmDescriptor.ObjectMeta{
					Name:    vector.Reference.Component,
					Version: vector.Version,
				},
			},
			RepositoryContexts: nil,
			Provider: ocmDescriptor.Provider{
				Name: "konfidence", // TODO: How to set this properly?
			},
			Resources:  nil,
			Sources:    nil,
			References: latestArtifacts,
		},
	}
}

func mapToReference(artifact domain.Artifact, hashAlgo, normAlgo string) ocmDescriptor.Reference {
	return ocmDescriptor.Reference{
		ElementMeta: ocmDescriptor.ElementMeta{
			ObjectMeta: ocmDescriptor.ObjectMeta{
				Name:    createReferenceName(artifact),
				Version: artifact.Version,
			},
		},
		Component: artifact.OcmReference.Component,
		Digest: ocmDescriptor.Digest{
			HashAlgorithm:          hashAlgo,
			NormalisationAlgorithm: normAlgo,
			Value:                  artifact.Digest,
		},
	}
}

func createReferenceName(artifact domain.Artifact) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(artifact.OcmReference.Component))
	sum := h.Sum64()
	return strconv.FormatUint(sum, 36)
}

// NewAdapter creates a new OCM Adapter with the given options.
func NewAdapter(options ...AdapterOption) Adapter {
	a := Adapter{}
	for _, opt := range options {
		opt(&a)
	}
	applyDefaults(&a)
	return a
}

func applyDefaults(a *Adapter) {
	if a.vectorSigner == nil {
		ctrl.Log.Info("vector signer not configured - using noop signer")
		a.vectorSigner = NoopSigner{}
	}
	if a.digester == nil {
		a.digester = newOcmDigester()
	}
	if a.vectorVerifier == nil {
		ctrl.Log.Info("vector verifier not configured - using noop verifier")
		a.vectorVerifier = NoopVerifier{}
	}
	if a.artifactVerifier == nil {
		ctrl.Log.Info("artifact verifier not configured - using noop verifier")
		a.artifactVerifier = NoopVerifier{}
	}
}
