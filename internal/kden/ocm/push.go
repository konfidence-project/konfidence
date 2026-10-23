package ocm

import (
	"context"
	"fmt"

	"github.com/konfidence-project/konfidence/internal/kden/log"
	ocmgenericspecv1 "ocm.software/open-component-model/bindings/go/configuration/generic/v1/spec"
	ocmconstructor "ocm.software/open-component-model/bindings/go/constructor"
	ocmconstructorruntime "ocm.software/open-component-model/bindings/go/constructor/runtime"
	ocmconstructorspecv1 "ocm.software/open-component-model/bindings/go/constructor/spec/v1"
	ocmoci "ocm.software/open-component-model/bindings/go/oci"
	ocmaddcvcli "ocm.software/open-component-model/cli/cmd/add/component-version"
)

func GetOcmConstructorProvider(ocmConfiguration *ocmgenericspecv1.Config, ctx context.Context,
	registry string) (*ConstructorProvider, error) {
	pluginManager, err := ocmGetPluginManager(ctx, ocmConfiguration)
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin manager: %w", err)
	}

	repoSpec, err := ocmGetRepositorySpec(registry)
	if err != nil {
		return nil, fmt.Errorf("failed to load repository spec: %w", err)
	}

	credentialsGraph, err := ocmGetCredentialGraph(ctx, pluginManager, ocmConfiguration)
	if err != nil {
		return nil, fmt.Errorf("failed to get credential graph: %w", err)
	}

	// Pass the credential graph to the resolver so resolving external component
	// references (e.g. a vector's componentReferences) is authenticated, not just
	// the target repository.
	repoResolver, err := ocmGetComponentRepositoryResolver(
		ctx,
		pluginManager.ComponentVersionRepositoryRegistry,
		credentialsGraph,
		WithRepository(repoSpec),
		WithConfig(ocmConfiguration))
	if err != nil {
		return nil, fmt.Errorf("failed to get repository resolver: %w", err)
	}

	return &ConstructorProvider{
		Cache:              "",
		TargetRepoSpec:     repoSpec,
		RepositoryResolver: repoResolver,
		PluginManager:      pluginManager,
		Graph:              credentialsGraph,
	}, nil
}

func PushComponentConstructor(ctx context.Context, constructorOptionProvider *ConstructorProvider,
	constructor *ocmconstructorspecv1.ComponentConstructor) error {
	opts := ocmconstructor.Options{
		TargetRepositoryProvider:            constructorOptionProvider,
		ResourceRepositoryProvider:          constructorOptionProvider,
		SourceInputMethodProvider:           constructorOptionProvider,
		ResourceInputMethodProvider:         constructorOptionProvider,
		ExternalComponentRepositoryProvider: constructorOptionProvider,
		ResourceDigestProcessorProvider:     constructorOptionProvider,
		Resolver:                            constructorOptionProvider.Graph,
		ConcurrencyLimit:                    1,
		ComponentVersionConflictPolicy:      ocmaddcvcli.ComponentVersionConflictPolicy("abort-and-fail").ToConstructorConflictPolicy(),
	}

	runtimeConstructor := ocmConvertToRuntimeConstructor(constructor)
	if runtimeConstructor == nil {
		return fmt.Errorf("failed to create runtime constructor")
	}
	opts.ExternalComponentVersionCopyPolicy = externalCopyPolicy(ctx, constructorOptionProvider, runtimeConstructor)
	componentConstructor := ocmCreateConstructor(runtimeConstructor, opts)
	return componentConstructor.Construct(ctx)
}

// externalCopyPolicy skips copying the referenced components when all of them
// already exist in the target repository, as for a vector pushed next to its
// components. That copy only re-uploads identical blobs, and OCM reads those
// blobs with a context that discovery has already cancelled, which fails
// intermittently (#414). References into other repositories keep copy-or-fail,
// so the target ends up self-contained.
func externalCopyPolicy(ctx context.Context, provider *ConstructorProvider,
	constructor *ocmconstructorruntime.ComponentConstructor) ocmconstructor.ExternalComponentVersionCopyPolicy {
	refs := externalReferences(constructor)
	if len(refs) == 0 {
		return ocmconstructor.ExternalComponentVersionCopyPolicyCopyOrFail
	}
	repo, err := ocmGetTargetRepository(ctx, nil, provider)
	if err != nil {
		log.Debug("copying referenced components: target repository unavailable", "error", err)
		return ocmconstructor.ExternalComponentVersionCopyPolicyCopyOrFail
	}
	for _, ref := range refs {
		if _, err := repo.GetComponentVersion(ctx, ref.Component, ref.Version); err != nil {
			log.Debug("copying referenced components: not in target repository",
				"component", ref.Component, "version", ref.Version, "error", err)
			return ocmconstructor.ExternalComponentVersionCopyPolicyCopyOrFail
		}
	}
	return ocmconstructor.ExternalComponentVersionCopyPolicySkip
}

// externalReferences returns the references to components that the
// constructor does not build itself.
func externalReferences(constructor *ocmconstructorruntime.ComponentConstructor) []ocmconstructorruntime.Reference {
	built := make(map[string]bool, len(constructor.Components))
	for _, c := range constructor.Components {
		built[c.Name+":"+c.Version] = true
	}
	var refs []ocmconstructorruntime.Reference
	for _, c := range constructor.Components {
		for _, ref := range c.References {
			if !built[ref.Component+":"+ref.Version] {
				refs = append(refs, ref)
			}
		}
	}
	return refs
}

func AddComponentVersionAlias(ctx context.Context, component, versionOrAlias, alias string,
	constructorOptionsProvider *ConstructorProvider) error {
	repo, err := ocmGetTargetRepository(ctx, nil, constructorOptionsProvider)
	if err != nil {
		return fmt.Errorf("failed to get target repository")
	}

	aliasRepo, ok := repo.(ocmoci.AliasComponentVersionRepository)
	if !ok {
		return fmt.Errorf("failed to get repository that supports aliasing")
	}

	return aliasRepo.AddComponentVersionAlias(ctx, component, versionOrAlias, alias)
}
