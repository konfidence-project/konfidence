package ocm

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"strconv"

	"github.com/konfidence-project/pkg/ocm/crypto"
	pkgOcm "github.com/konfidence-project/pkg/ocm/repository"
	ocmDescriptor "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	"ocm.software/open-component-model/bindings/go/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/konfidence-project/gcp-vector-assembly-controller/internal/controller/domain"
)

var (
	_ domain.VectorOcmPort = (*Adapter)(nil)
)

// Adapter is an implementation of the VectorOcmPort interface that interacts with OCM repositories to manage vectors and their associated artifacts.
type Adapter struct {
	vectorVerifier, artifactVerifier crypto.Verifier
	vectorSigner                     crypto.Signer
	digester                         crypto.Digester
	ocmClient                        pkgOcm.Client
}

func (a Adapter) GetLatestArtifactVersions(ctx context.Context, references []compref.Ref) ([]domain.Artifact, error) {
	if len(references) == 0 {
		return nil, nil
	}

	artifacts, unverifiedDescriptors := make([]domain.Artifact, 0, len(references)), make([]*ocmDescriptor.Descriptor, 0, len(references))
	for _, ref := range references {
		desc, err := a.ocmClient.Get(ctx, ref)
		if err != nil {
			return nil,
				fmt.Errorf("unable to get latest descriptor for ocm component (%s): %w", ref, err)
		}
		unverifiedDescriptors = append(unverifiedDescriptors, &desc)
		dig, err := a.digester.GenerateDigest(ctx, &desc)
		if err != nil {
			return nil, fmt.Errorf("unable to generate digest for ocm descriptor for component (%s) version (%s): %w", ref, desc.Component.Version, err)
		}
		artifact := domain.Artifact{
			Version:    desc.Component.Version,
			Name:       desc.Component.Name,
			Digest:     dig.Value,
			SourceRepo: ref.Repository,
		}
		artifacts = append(artifacts, artifact)
	}
	if err := a.artifactVerifier.Verify(ctx, unverifiedDescriptors...); err != nil {
		return nil, fmt.Errorf("ocm artifact verification failed for one or more artifacts: %w", err)
	}
	return artifacts, nil
}

func (a Adapter) GetLatestVector(ctx context.Context, vectorRef compref.Ref) (domain.Vector, error) {
	descriptor, err := a.ocmClient.Get(ctx, vectorRef)
	if err != nil {
		return domain.Vector{}, fmt.Errorf("unable to get latest ocm descriptor for vector (%s): %w", vectorRef, err)
	}
	if err := a.vectorVerifier.Verify(ctx, &descriptor); err != nil {
		return domain.Vector{}, fmt.Errorf("unable to verify ocm descriptor for vector (%s) version (%s): %w",
			vectorRef, descriptor.Component.Version, err)
	}
	return mapToDomain(descriptor, vectorRef.Repository), nil
}

func (a Adapter) CreateVector(ctx context.Context, repoSpec runtime.Typed, vector domain.Vector) error {
	if err := a.copyArtifacts(ctx, vector.Artifacts, repoSpec); err != nil {
		return fmt.Errorf("unable to copy artifact components to target repository: %w", err)
	}

	vectorDescriptor := mapToDescriptor(vector, a.digester.GetHashAlgorithm().String(), a.digester.GetNormalisationAlgorithm())
	if err := a.vectorSigner.Sign(ctx, &vectorDescriptor); err != nil {
		return fmt.Errorf("unable to Sign ocm descriptor for vector (%s): %w", vector.Name, err)
	}

	if err := a.ocmClient.Save(ctx, repoSpec, vectorDescriptor); err != nil {
		if errors.Is(err, pkgOcm.ErrComponentAlreadyExists) {
			return nil
		}
		return fmt.Errorf("unable to save ocm descriptor for vector (%s) version (%s): %w", vector.Name, vector.Version, err)
	}

	return nil
}

func mapToDomain(descriptor ocmDescriptor.Descriptor, vectorRepository runtime.Typed) domain.Vector {

	artifacts := make([]domain.Artifact, 0, len(descriptor.Component.References))
	for _, ref := range descriptor.Component.References {
		artifact := domain.Artifact{
			Version:    ref.Version,
			Name:       ref.Component,
			Digest:     ref.Digest.Value,
			SourceRepo: vectorRepository,
		}
		artifacts = append(artifacts, artifact)
	}
	return domain.Vector{
		Version:   descriptor.Component.Version,
		Name:      descriptor.Component.Name,
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
					Name:    vector.Name,
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
		Component: artifact.Name,
		Digest: ocmDescriptor.Digest{
			HashAlgorithm:          hashAlgo,
			NormalisationAlgorithm: normAlgo,
			Value:                  artifact.Digest,
		},
	}
}

func createReferenceName(artifact domain.Artifact) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(artifact.Name))
	sum := h.Sum64()
	return strconv.FormatUint(sum, 36)
}

func (a Adapter) copyArtifacts(ctx context.Context, artifacts []domain.Artifact, targetRepoSpec runtime.Typed) error {
	artifactReferences := make([]compref.Ref, 0, len(artifacts))

	for _, artifact := range artifacts {
		sourceRef := compref.Ref{
			Repository: artifact.SourceRepo,
			Component:  artifact.Name,
			Version:    artifact.Version,
		}
		artifactReferences = append(artifactReferences, sourceRef)
	}

	if err := a.ocmClient.Copy(ctx, artifactReferences, targetRepoSpec); err != nil {
		return fmt.Errorf("failed to copy artifacts: %w", err)
	}
	return nil
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
		a.vectorSigner = crypto.NoopSigner{}
	}
	if a.digester == nil {
		a.digester = crypto.NewDefaultDigester(ctrl.Log)
	}
	if a.vectorVerifier == nil {
		ctrl.Log.Info("vector verifier not configured - using noop verifier")
		a.vectorVerifier = crypto.NoopVerifier{}
	}
	if a.artifactVerifier == nil {
		ctrl.Log.Info("artifact verifier not configured - using noop verifier")
		a.artifactVerifier = crypto.NoopVerifier{}
	}
}
