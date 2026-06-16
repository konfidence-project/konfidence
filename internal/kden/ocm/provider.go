package ocm

import (
	"context"
	"errors"
	"fmt"

	"github.com/konfidence-project/konfidence/internal/kden/log"
	"ocm.software/open-component-model/bindings/go/blob"
	"ocm.software/open-component-model/bindings/go/constructor"
	constructorruntime "ocm.software/open-component-model/bindings/go/constructor/runtime"
	ocmcredentials "ocm.software/open-component-model/bindings/go/credentials"
	descriptorruntime "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	"ocm.software/open-component-model/bindings/go/plugin/manager"
	ocmpluginmanagerresource "ocm.software/open-component-model/bindings/go/plugin/manager/registries/resource"
	"ocm.software/open-component-model/bindings/go/repository"
	"ocm.software/open-component-model/bindings/go/repository/component/resolvers"
	"ocm.software/open-component-model/bindings/go/runtime"
)

type ConstructorProvider struct {
	Cache              string
	TargetRepoSpec     runtime.Typed
	RepositoryResolver resolvers.ComponentVersionRepositoryResolver
	PluginManager      *manager.PluginManager
	Graph              ocmcredentials.Resolver
}

type constructorPlugin struct {
	plugin ocmpluginmanagerresource.Repository
}

func (prov *ConstructorProvider) GetExternalRepository(ctx context.Context, name, version string) (repository.ComponentVersionRepository, error) {
	if prov.RepositoryResolver == nil {
		return nil, fmt.Errorf("failed to fetch external component version %s:%s repository provider configured", name, version)
	}
	return prov.RepositoryResolver.GetComponentVersionRepositoryForComponent(ctx, name, version)
}

func (prov *ConstructorProvider) GetDigestProcessor(ctx context.Context, resource *descriptorruntime.Resource) (constructor.ResourceDigestProcessor, error) {
	return prov.PluginManager.DigestProcessorRegistry.GetPlugin(ctx, resource.Access)
}

func (prov *ConstructorProvider) GetResourceInputMethod(ctx context.Context, resource *constructorruntime.Resource) (constructor.ResourceInputMethod, error) {
	return prov.PluginManager.InputRegistry.GetResourceInputPlugin(ctx, resource.Input)
}

func (prov *ConstructorProvider) GetSourceInputMethod(ctx context.Context, src *constructorruntime.Source) (constructor.SourceInputMethod, error) {
	return prov.PluginManager.InputRegistry.GetSourceInputPlugin(ctx, src.Input)
}

func (prov *ConstructorProvider) GetResourceRepository(ctx context.Context, resource *constructorruntime.Resource) (constructor.ResourceRepository, error) {
	plugin, err := prov.PluginManager.ResourcePluginRegistry.GetResourcePlugin(ctx, resource.Access)
	if err != nil {
		return nil, fmt.Errorf("failed to get plugin for resource %q : %w", resource.Access, err)
	}
	return &constructorPlugin{plugin: plugin}, nil
}

func (c *constructorPlugin) GetResourceCredentialConsumerIdentity(
	ctx context.Context, resource *constructorruntime.Resource,
) (identity runtime.Identity, err error) {
	return c.plugin.GetResourceCredentialConsumerIdentity(ctx, constructorruntime.ConvertToDescriptorResource(resource))
}

func (c *constructorPlugin) DownloadResource(
	ctx context.Context, res *descriptorruntime.Resource, credentials runtime.Typed,
) (content blob.ReadOnlyBlob, err error) {
	return c.plugin.DownloadResource(ctx, res, credentials)
}

func (prov *ConstructorProvider) GetTargetRepository(ctx context.Context, _ *constructorruntime.Component) (constructor.TargetRepository, error) {
	var creds runtime.Typed
	identity, err := prov.PluginManager.ComponentVersionRepositoryRegistry.GetComponentVersionRepositoryCredentialConsumerIdentity(ctx, prov.TargetRepoSpec)
	if err == nil {
		if prov.Graph != nil {
			if creds, err = prov.Graph.Resolve(ctx, identity); err != nil { //nolint:staticcheck //this is what ocm currently use
				if errors.Is(err, ocmcredentials.ErrNotFound) {
					log.Debug(fmt.Sprintf("failed to resolve credentials for repository %q: %s", prov.TargetRepoSpec, err.Error()))
				} else {
					return nil, fmt.Errorf("failed to resolve credentials for repository %q: %w", prov.TargetRepoSpec, err)
				}
			}
		}
	} else {
		log.Debug("failed to get credential consumer identity for component version repository", "repository", prov.TargetRepoSpec, "error", err)
	}

	return prov.PluginManager.ComponentVersionRepositoryRegistry.GetComponentVersionRepository(ctx, prov.TargetRepoSpec, creds)
}
