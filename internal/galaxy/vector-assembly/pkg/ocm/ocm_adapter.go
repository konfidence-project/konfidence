package ocm

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"maps"
	goruntime "runtime"
	"slices"
	"strconv"
	"sync"

	"github.com/konfidence-project/pkg/ocm/crypto"
	pkgocm "github.com/konfidence-project/pkg/ocm/repository"
	"golang.org/x/sync/errgroup"
	ocmDescriptor "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	"ocm.software/open-component-model/bindings/go/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/konfidence-project/gcp-vector-assembly-controller/internal/controller/domain"
)

var (
	_ domain.VectorOcmPort = (*Adapter)(nil)
)

// Adapter is an implementation of the VectorOcmPort interface that interacts
// with OCM repositories to manage vectors and their associated artifacts.
type Adapter struct {
	vectorVerifier, artifactVerifier crypto.Verifier
	vectorSigner                     crypto.Signer
	digester                         crypto.Digester
	ocmClient                        pkgocm.Client
}

func (a Adapter) GetArtifacts(ctx context.Context, references []compref.Ref) ([]domain.Artifact, error) {
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

func (a Adapter) fetchAndCollectDescriptors(ctx context.Context, references []compref.Ref) (map[compref.Ref]*ocmDescriptor.Descriptor, error) {
	var (
		mux    = sync.Mutex{}
		result = make(map[compref.Ref]*ocmDescriptor.Descriptor, len(references))
	)
	if len(references) == 1 {
		if err := a.fetchAndCollectDescriptor(ctx, &mux, references[0], result); err != nil {
			return nil, err
		}
		return result, nil
	}
	pool, ctx2 := errgroup.WithContext(ctx)
	pool.SetLimit(min(32, len(references))) // limit to keep parallel requests bounded
	for _, ref := range references {
		pool.Go(func() error { return a.fetchAndCollectDescriptor(ctx2, &mux, ref, result) })
	}
	if err := pool.Wait(); err != nil {
		return nil, err
	}
	return result, nil
}

func (a Adapter) fetchAndCollectDescriptor(
	ctx context.Context, mux *sync.Mutex, ref compref.Ref, results map[compref.Ref]*ocmDescriptor.Descriptor) error {
	desc, err := a.ocmClient.Get(ctx, ref)
	if err != nil {
		return fmt.Errorf("unable to get descriptor for ocm component (%s): %w", ref, err)
	}
	mux.Lock()
	results[ref] = &desc
	mux.Unlock()
	return nil
}

func (a Adapter) digestAndCollectArtifacts(ctx context.Context, references map[compref.Ref]*ocmDescriptor.Descriptor) ([]domain.Artifact, error) {
	results := make([]domain.Artifact, len(references))
	keys := slices.Collect(maps.Keys(references))
	if len(references) == 1 {
		key := slices.Collect(maps.Keys(references))[0]
		value := slices.Collect(maps.Values(references))[0]
		if err := a.digestAndCollectSingleArtifact(ctx, 0, key, value, results); err != nil {
			return nil, err
		}
		return results, nil
	}
	pool, ctx2 := errgroup.WithContext(ctx)
	pool.SetLimit(min(goruntime.GOMAXPROCS(0), len(references))) // no oversubscription on CPU bound digest generation tasks
	for i, ref := range keys {
		pool.Go(func() error {
			return a.digestAndCollectSingleArtifact(ctx2, i, ref, references[ref], results)
		})
	}
	if err := pool.Wait(); err != nil {
		return nil, err
	}
	return results, nil
}

func (a Adapter) digestAndCollectSingleArtifact(
	ctx context.Context,
	index int,
	ref compref.Ref,
	desc *ocmDescriptor.Descriptor,
	results []domain.Artifact) error {
	dig, err := a.digester.GenerateDigest(ctx, desc)
	if err != nil {
		return fmt.Errorf("unable to generate digest for ocm descriptor (%s) version (%s): %w", ref, desc.Component.Version, err)
	}
	results[index] = domain.Artifact{
		Version:    desc.Component.Version,
		Name:       desc.Component.Name,
		Digest:     dig.Value,
		SourceRepo: ref.Repository,
	}
	return nil
}

func (a Adapter) GetVector(ctx context.Context, vectorRef compref.Ref) (domain.Vector, error) {
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

func (a Adapter) CreateVector(ctx context.Context, repoSpec runtime.Typed, vector domain.Vector, alias string) error {
	if err := a.copyArtifacts(ctx, vector.Artifacts, repoSpec); err != nil {
		return fmt.Errorf("unable to copy artifact components to target repository: %w", err)
	}
	vectorDescriptor := mapToDescriptor(vector, a.digester.GetHashAlgorithm().String(), a.digester.GetNormalisationAlgorithm())
	if err := a.vectorSigner.Sign(ctx, &vectorDescriptor); err != nil {
		return fmt.Errorf("unable to Sign ocm descriptor for vector (%s): %w", vector.Name, err)
	}

	if err := a.ocmClient.Save(ctx, repoSpec, vectorDescriptor); err != nil {
		if errors.Is(err, pkgocm.ErrComponentAlreadyExists) {
			return nil
		}
		return fmt.Errorf("unable to save ocm descriptor for vector (%s) version (%s): %w", vector.Name, vector.Version, err)
	}
	aliasRef := compref.Ref{
		Repository: repoSpec,
		Component:  vectorDescriptor.Component.Name,
		Version:    vectorDescriptor.Component.Version,
	}
	if err := a.ocmClient.AddAlias(ctx, aliasRef, alias); err != nil {
		return fmt.Errorf("unable to add alias (%s) for vector (%s) version (%s): %w", alias, vector.Name, vector.Version, err)
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

// NewPortProvider creates a VectorOcmPortProviderFunc that builds an Adapter
// with the given options and plugs in the provided client at call time.
func NewPortProvider(opts ...AdapterOption) domain.VectorOcmPortProviderFunc {
	return func(client pkgocm.Client) domain.VectorOcmPort {
		a := NewAdapter(opts...)
		a.ocmClient = client
		return a
	}
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
