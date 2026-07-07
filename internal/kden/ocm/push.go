package ocm

import (
	"context"
	"fmt"

	ocmgenericspecv1 "ocm.software/open-component-model/bindings/go/configuration/generic/v1/spec"
	ocmconstructor "ocm.software/open-component-model/bindings/go/constructor"
	ocmconstructorspecv1 "ocm.software/open-component-model/bindings/go/constructor/spec/v1"
	ocmoci "ocm.software/open-component-model/bindings/go/oci"
	ocmaddcvcli "ocm.software/open-component-model/cli/cmd/add/component-version"
)

func GetOcmConstructorProvider(ocmConfiguration *ocmgenericspecv1.Config, ctx context.Context,
	registry string) (*ConstructorProvider, error) {
	pluginManager, err := ocmGetPluginManager(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin manager: %w", err)
	}

	repoSpec, err := ocmGetRepositorySpec(registry)
	if err != nil {
		return nil, fmt.Errorf("failed to load repository spec: %w", err)
	}

	repoResolver, err := ocmGetComponentRepositoryResolver(
		ctx,
		pluginManager.ComponentVersionRepositoryRegistry,
		nil,
		WithRepository(repoSpec),
		WithConfig(ocmConfiguration))
	if err != nil {
		return nil, fmt.Errorf("failed to get repository resolver: %w", err)
	}

	credentialsGraph, err := ocmGetCredentialGraph(ctx, pluginManager, ocmConfiguration)
	if err != nil {
		return nil, fmt.Errorf("failed to get credential graph: %w", err)
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
		Resolver:                            constructorOptionProvider.Graph,
		ConcurrencyLimit:                    1,
		ComponentVersionConflictPolicy:      ocmaddcvcli.ComponentVersionConflictPolicy("abort-and-fail").ToConstructorConflictPolicy(),
		ExternalComponentVersionCopyPolicy:  ocmaddcvcli.ExternalComponentVersionCopyPolicy("copy-or-fail").ToConstructorPolicy(),
	}

	runtimeConstructor := ocmConvertToRuntimeConstructor(constructor)
	if runtimeConstructor == nil {
		return fmt.Errorf("failed to create runtime constructor")
	}
	componentConstructor := ocmCreateConstructor(runtimeConstructor, opts)
	return componentConstructor.Construct(ctx)
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
