package ocm

import (
	"context"
	"fmt"

	ocmgenericspecv1 "ocm.software/open-component-model/bindings/go/configuration/generic/v1/spec"
	ocmconstructor "ocm.software/open-component-model/bindings/go/constructor"
	ocmconstructorspecv1 "ocm.software/open-component-model/bindings/go/constructor/spec/v1"
	ocmaddcvcli "ocm.software/open-component-model/cli/cmd/add/component-version"
)

func PushComponentConstructor(ocmConfiguration *ocmgenericspecv1.Config, ctx context.Context, registry string,
	constructor *ocmconstructorspecv1.ComponentConstructor) error {
	pluginManager, err := ocmGetPluginManager(ctx)
	if err != nil {
		return fmt.Errorf("failed to load plugin manager: %w", err)
	}

	repoSpec, err := ocmGetRepositorySpec(registry)
	if err != nil {
		return fmt.Errorf("failed to load repository spec: %w", err)
	}

	repoResolver, err := ocmGetComponentRepositoryResolver(
		ctx,
		pluginManager.ComponentVersionRepositoryRegistry,
		nil,
		WithRepository(repoSpec),
		WithConfig(ocmConfiguration))
	if err != nil {
		return fmt.Errorf("failed to get repository resolver: %w", err)
	}

	credentialsGraph, err := ocmGetCredentialGraph(ctx, pluginManager, ocmConfiguration)
	if err != nil {
		return fmt.Errorf("failed to get credential graph: %w", err)
	}

	constructorOptionProvider := &ConstructorProvider{
		Cache:              "",
		TargetRepoSpec:     repoSpec,
		RepositoryResolver: repoResolver,
		PluginManager:      pluginManager,
		Graph:              credentialsGraph,
	}
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
