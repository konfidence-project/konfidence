package ocm

import (
	"context"
	"fmt"

	ocmgenericspecv1 "ocm.software/open-component-model/bindings/go/configuration/generic/v1/spec"
	ocmconstructor "ocm.software/open-component-model/bindings/go/constructor"
	ocmconstructorruntime "ocm.software/open-component-model/bindings/go/constructor/runtime"
	ocmconstructorspecv1 "ocm.software/open-component-model/bindings/go/constructor/spec/v1"
	"ocm.software/open-component-model/bindings/go/credentials"
	"ocm.software/open-component-model/bindings/go/plugin/manager"
	"ocm.software/open-component-model/bindings/go/repository"
	"ocm.software/open-component-model/bindings/go/repository/component/resolvers"
	"ocm.software/open-component-model/bindings/go/runtime"
	ocmaddcvcli "ocm.software/open-component-model/cli/cmd/add/component-version"
)

type OcmMethods struct {
	GetPluginManager               func(ctx context.Context) (*manager.PluginManager, error)
	GetRepositorySpec              func(repository string) (runtime.Typed, error)
	GetComponentRepositoryResolver func(
		ctx context.Context,
		repoProvider repository.ComponentVersionRepositoryProvider,
		credentialGraph credentials.Resolver,
		opts ...RepositoryResolverOption,
	) (resolvers.ComponentVersionRepositoryResolver, error)
	GetCredentialGraph          func(pluginManager *manager.PluginManager, ctx context.Context, config *ocmgenericspecv1.Config) (credentials.Resolver, error)
	ConvertToRuntimeConstructor func(constructor *ocmconstructorspecv1.ComponentConstructor) *ocmconstructorruntime.ComponentConstructor
	CreateConstructor           func(constructor *ocmconstructorruntime.ComponentConstructor, opts ocmconstructor.Options) ocmconstructor.Constructor
}

var OcmMethodsImpl = OcmMethods{
	GetPluginManager:               GetPluginManager,
	GetRepositorySpec:              GetRepositorySpec,
	GetComponentRepositoryResolver: GetComponentRepositoryResolver,
	GetCredentialGraph:             GetCredentialGraph,
	ConvertToRuntimeConstructor:    ocmconstructorruntime.ConvertToRuntimeConstructor,
	CreateConstructor:              ocmconstructor.NewDefaultConstructor,
}

func PushComponentConstructor(
	ocmConfiguration *ocmgenericspecv1.Config,
	ctx context.Context,
	registry string,
	constructor *ocmconstructorspecv1.ComponentConstructor,
) error {
	pluginManager, err := OcmMethodsImpl.GetPluginManager(ctx)
	if err != nil {
		return fmt.Errorf("failed to load plugin manager: %w", err)
	}

	repoSpec, err := OcmMethodsImpl.GetRepositorySpec(registry)
	if err != nil {
		return fmt.Errorf("failed to load repository spec: %w", err)
	}

	repoResolver, err := OcmMethodsImpl.GetComponentRepositoryResolver(
		ctx,
		pluginManager.ComponentVersionRepositoryRegistry,
		nil,
		WithRepository(repoSpec),
		WithConfig(ocmConfiguration))
	if err != nil {
		return fmt.Errorf("failed to get repository resolver: %w", err)
	}

	credentialsGraph, err := OcmMethodsImpl.GetCredentialGraph(pluginManager, ctx, ocmConfiguration)
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
		ComponentVersionConflictPolicy: ocmaddcvcli.ComponentVersionConflictPolicy("abort-and-fail").
			ToConstructorConflictPolicy(),
		ExternalComponentVersionCopyPolicy: ocmaddcvcli.ExternalComponentVersionCopyPolicy("copy-or-fail").
			ToConstructorPolicy(),
	}

	runtimeConstructor := OcmMethodsImpl.ConvertToRuntimeConstructor(constructor)
	if runtimeConstructor == nil {
		return fmt.Errorf("failed to create runtime constructor")
	}
	componentConstructor := OcmMethodsImpl.CreateConstructor(runtimeConstructor, opts)
	return componentConstructor.Construct(ctx)
}
