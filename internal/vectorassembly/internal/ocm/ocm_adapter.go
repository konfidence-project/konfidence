package ocm

//go:generate go run go.uber.org/mock/mockgen -destination=mocks/mock_client.go -package=mocks github.com/konfidence-project/konfidence/pkg/ocm/repository Client
//go:generate go run go.uber.org/mock/mockgen -destination=mocks/mock_verifier.go -package=mocks github.com/konfidence-project/konfidence/pkg/ocm/crypto Verifier
//go:generate go run go.uber.org/mock/mockgen -destination=mocks/mock_signer.go -package=mocks github.com/konfidence-project/konfidence/pkg/ocm/crypto Signer
//go:generate go run go.uber.org/mock/mockgen -destination=mocks/mock_digester.go -package=mocks github.com/konfidence-project/konfidence/pkg/ocm/crypto Digester

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"maps"
	"slices"
	"strconv"
	"sync"

	"github.com/konfidence-project/konfidence/internal/vectorassembly/internal/vector"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	pkgocm "github.com/konfidence-project/konfidence/pkg/ocm/repository"
	"github.com/opencontainers/go-digest"
	"golang.org/x/sync/errgroup"
	"ocm.software/open-component-model/bindings/go/blob"
	"ocm.software/open-component-model/bindings/go/blob/inmemory"
	ocmdescriptor "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	v2 "ocm.software/open-component-model/bindings/go/descriptor/v2"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	"ocm.software/open-component-model/bindings/go/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	DefaultVectorConfigName    = "cloud-konfidence-vector-config"
	DefaultVectorConfigVersion = "1.0.0"
)

var (
	_ vector.OcmPort = (*Adapter)(nil)
)

// Adapter is an implementation of the VectorOcmPort interface that interacts
// with OCM repositories to manage vectors and their associated artifacts.
type Adapter struct {
	vectorVerifier, artifactVerifier crypto.Verifier
	vectorSigner                     crypto.Signer
	digester                         crypto.Digester
	ocmClient                        pkgocm.Client
}

func (a Adapter) GetArtifacts(ctx context.Context, references []compref.Ref) ([]vector.Artifact, error) {
	if len(references) == 0 {
		return nil, nil
	}
	unverifiedResults, err := a.fetchAndCollectDescriptors(ctx, references)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch descriptors for artifact references: %w", err)
	}
	if err := a.artifactVerifier.Verify(ctx, slices.Collect(maps.Values(unverifiedResults))...); err != nil {
		return nil, fmt.Errorf("ocm artifact verification failed for one or more artifacts: %w", err)
	}
	artifacts, err := a.digestAndCollectArtifacts(ctx, unverifiedResults)
	if err != nil {
		return nil, fmt.Errorf("failed to digest artifact references: %w", err)
	}
	return artifacts, nil
}

func (a Adapter) fetchAndCollectDescriptors(ctx context.Context, references []compref.Ref) (map[compref.Ref]*ocmdescriptor.Descriptor, error) {
	var (
		mux    = sync.Mutex{}
		result = make(map[compref.Ref]*ocmdescriptor.Descriptor, len(references))
	)
	if len(references) == 1 {
		if err := a.fetchAndCollectDescriptor(ctx, &mux, references[0], result); err != nil {
			return nil, err
		}
		return result, nil
	}
	pool, ctx2 := errgroup.WithContext(ctx)
	pool.SetLimit(min(128, len(references))) // limit to keep parallel requests bounded
	for _, ref := range references {
		pool.Go(func() error { return a.fetchAndCollectDescriptor(ctx2, &mux, ref, result) })
	}
	if err := pool.Wait(); err != nil {
		return nil, err
	}
	return result, nil
}

func (a Adapter) fetchAndCollectDescriptor(
	ctx context.Context, mux *sync.Mutex, ref compref.Ref, results map[compref.Ref]*ocmdescriptor.Descriptor) error {
	desc, err := a.ocmClient.Get(ctx, ref)
	if err != nil {
		return fmt.Errorf("unable to get descriptor for ocm component (%s): %w", ref, err)
	}
	mux.Lock()
	results[ref] = &desc
	mux.Unlock()
	return nil
}

func (a Adapter) digestAndCollectArtifacts(ctx context.Context, references map[compref.Ref]*ocmdescriptor.Descriptor) ([]vector.Artifact, error) {
	results := make([]vector.Artifact, 0, len(references))
	for ref, desc := range references {
		dig, err := a.digester.GenerateDigest(ctx, desc)
		if err != nil {
			return nil, fmt.Errorf("unable to generate digest for ocm descriptor (%s) version (%s): %w", ref, desc.Component.Version, err)
		}
		results = append(results, vector.Artifact{
			Version:    desc.Component.Version,
			Name:       desc.Component.Name,
			Digest:     dig.Value,
			SourceRepo: ref.Repository,
		})
	}
	return results, nil
}

func (a Adapter) GetVector(ctx context.Context, vectorRef compref.Ref) (vector.Vector, error) {
	descriptor, err := a.ocmClient.Get(ctx, vectorRef)
	if err != nil {
		return vector.Vector{}, fmt.Errorf("unable to get latest ocm descriptor for vector (%s): %w", vectorRef, err)
	}
	if err := a.vectorVerifier.Verify(ctx, &descriptor); err != nil {
		return vector.Vector{}, fmt.Errorf("unable to verify ocm descriptor for vector (%s) version (%s): %w",
			vectorRef, descriptor.Component.Version, err)
	}

	// get vector configuration if existent
	vectorConfiguration, err := a.getVectorConfiguration(ctx, vectorRef, descriptor)
	if err != nil {
		return vector.Vector{}, err
	}

	return mapToDomain(descriptor, vectorRef.Repository, vectorConfiguration), nil
}

func (a Adapter) getVectorConfiguration(ctx context.Context, vectorRef compref.Ref, descriptor ocmdescriptor.Descriptor) (*vector.VectorConfiguration, error) {
	var vectorConfigResource *ocmdescriptor.Resource
	for _, res := range descriptor.Component.Resources {
		if res.Name == DefaultVectorConfigName {
			vectorConfigResource = &res
			break
		}
	}

	var vectorConfiguration *vector.VectorConfiguration
	if vectorConfigResource != nil {
		readBlob, _, err := a.ocmClient.GetLocalResource(ctx, vectorRef, vectorConfigResource.ToIdentity())
		if err != nil {
			return nil, fmt.Errorf("unable to get vector configuration for vector (%s) version (%s): %w",
				vectorRef, descriptor.Component.Version, err)
		}

		var content bytes.Buffer
		err = blob.Copy(&content, readBlob)
		if err != nil {
			return nil, fmt.Errorf("unable to read vector configuration for vector (%s) version (%s): %w",
				vectorRef, descriptor.Component.Version, err)
		}

		vectorConfiguration = &vector.VectorConfiguration{
			Content: content.Bytes(),
		}
	}

	return vectorConfiguration, nil
}

func (a Adapter) CreateVector(ctx context.Context, repoSpec runtime.Typed, v vector.Vector, alias string) error {
	if err := a.copyArtifacts(ctx, v.Artifacts, repoSpec); err != nil {
		return fmt.Errorf("unable to copy artifact components to target repository: %w", err)
	}

	vectorDescriptor := mapToDescriptor(v, a.digester.GetHashAlgorithm(), a.digester.GetNormalisationAlgorithm())
	if v.VectorConfig != nil {
		// upload vector configuration as local resource
		content := inmemory.New(bytes.NewReader(v.VectorConfig.Content))
		vectorConfigResource := mapToVectorConfigResource(v)
		updatedResource, err := a.ocmClient.AddLocalResource(ctx, repoSpec, vectorDescriptor, *vectorConfigResource, content)
		if err != nil {
			return fmt.Errorf("unable to add vector configuration as local resource for vector (%s): %w", v.Name, err)
		}

		// add updated resource to vector descriptor
		vectorDescriptor.Component.Resources = append(vectorDescriptor.Component.Resources, *updatedResource)
	}

	if err := a.vectorSigner.Sign(ctx, &vectorDescriptor); err != nil {
		return fmt.Errorf("unable to Sign ocm descriptor for vector (%s): %w", v.Name, err)
	}

	if err := a.ocmClient.Save(ctx, repoSpec, vectorDescriptor); err != nil {
		if errors.Is(err, pkgocm.ErrComponentAlreadyExists) {
			return nil
		}
		return fmt.Errorf("unable to save ocm descriptor for vector (%s) version (%s): %w", v.Name, v.Version, err)
	}
	aliasRef := compref.Ref{
		Repository: repoSpec,
		Component:  vectorDescriptor.Component.Name,
		Version:    vectorDescriptor.Component.Version,
	}
	if err := a.ocmClient.AddAlias(ctx, aliasRef, alias); err != nil {
		return fmt.Errorf("unable to add alias (%s) for vector (%s) version (%s): %w", alias, v.Name, v.Version, err)
	}
	return nil
}

func mapToDomain(descriptor ocmdescriptor.Descriptor, vectorRepository runtime.Typed, vectorConfig *vector.VectorConfiguration) vector.Vector {
	artifacts := make([]vector.Artifact, 0, len(descriptor.Component.References))
	for _, ref := range descriptor.Component.References {
		artifact := vector.Artifact{
			Version:    ref.Version,
			Name:       ref.Component,
			Digest:     ref.Digest.Value,
			SourceRepo: vectorRepository,
		}
		artifacts = append(artifacts, artifact)
	}
	return vector.Vector{
		Version:      descriptor.Component.Version,
		Name:         descriptor.Component.Name,
		Artifacts:    artifacts,
		VectorConfig: vectorConfig,
	}
}

func mapToDescriptor(v vector.Vector, hashAlgo, normAlgo string) ocmdescriptor.Descriptor {
	latestArtifacts := make([]ocmdescriptor.Reference, 0, len(v.Artifacts))
	for _, artifact := range v.Artifacts {
		latestArtifacts = append(latestArtifacts, mapToReference(artifact, hashAlgo, normAlgo))
	}

	return ocmdescriptor.Descriptor{
		Meta: ocmdescriptor.Meta{
			Version: "v2",
		},
		Component: ocmdescriptor.Component{
			ComponentMeta: ocmdescriptor.ComponentMeta{
				ObjectMeta: ocmdescriptor.ObjectMeta{
					Name:    v.Name,
					Version: v.Version,
				},
			},
			RepositoryContexts: nil,
			Provider: ocmdescriptor.Provider{
				Name: "konfidence", // TODO: How to set this properly?
			},
			Resources:  nil,
			Sources:    nil,
			References: latestArtifacts,
		},
	}
}

func mapToVectorConfigResource(v vector.Vector) *ocmdescriptor.Resource {
	return &ocmdescriptor.Resource{
		Relation: ocmdescriptor.LocalRelation,
		ElementMeta: ocmdescriptor.ElementMeta{
			ObjectMeta: ocmdescriptor.ObjectMeta{
				Name:    DefaultVectorConfigName,
				Version: DefaultVectorConfigVersion,
			},
		},
		Type: "json",
		Access: &v2.LocalBlob{
			LocalReference: digest.FromBytes(v.VectorConfig.Content).String(),
			MediaType:      "application/json",
		},
	}
}

func mapToReference(artifact vector.Artifact, hashAlgo, normAlgo string) ocmdescriptor.Reference {
	return ocmdescriptor.Reference{
		ElementMeta: ocmdescriptor.ElementMeta{
			ObjectMeta: ocmdescriptor.ObjectMeta{
				Name:    createReferenceName(artifact),
				Version: artifact.Version,
			},
		},
		Component: artifact.Name,
		Digest: ocmdescriptor.Digest{
			HashAlgorithm:          hashAlgo,
			NormalisationAlgorithm: normAlgo,
			Value:                  artifact.Digest,
		},
	}
}

func createReferenceName(artifact vector.Artifact) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(artifact.Name))
	sum := h.Sum64()
	return strconv.FormatUint(sum, 36)
}

func (a Adapter) copyArtifacts(ctx context.Context, artifacts []vector.Artifact, targetRepoSpec runtime.Typed) error {
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
