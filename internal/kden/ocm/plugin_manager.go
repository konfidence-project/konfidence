package ocm

import (
	"context"
	"fmt"
	"strings"

	"ocm.software/open-component-model/bindings/go/input/dir"
	"ocm.software/open-component-model/bindings/go/input/file"
	ocicredentials "ocm.software/open-component-model/bindings/go/oci/credentials"
	"ocm.software/open-component-model/bindings/go/oci/repository/provider"
	v1identity "ocm.software/open-component-model/bindings/go/oci/spec/identity/v1"
	ocirepository "ocm.software/open-component-model/bindings/go/oci/spec/repository"
	"ocm.software/open-component-model/bindings/go/plugin/manager"
	"ocm.software/open-component-model/bindings/go/plugin/manager/registries/componentversionrepository"
	"ocm.software/open-component-model/bindings/go/plugin/manager/registries/credentialrepository"
	"ocm.software/open-component-model/bindings/go/plugin/manager/registries/digestprocessor"
	"ocm.software/open-component-model/bindings/go/plugin/manager/registries/input"
	"ocm.software/open-component-model/bindings/go/plugin/manager/registries/signinghandler"
	"ocm.software/open-component-model/bindings/go/rsa/signing/handler"
	signingv1alpha1 "ocm.software/open-component-model/bindings/go/rsa/signing/v1alpha1"
	ocmruntime "ocm.software/open-component-model/bindings/go/runtime"
)

type pluginManagerRegistrar interface {
	registerComponentVersionRepositoryPlugin(p componentversionrepository.BuiltinComponentVersionRepositoryProvider) error
	registerResourceInputPlugin(p input.BuiltinResourceInputMethod) error
	registerSourceInputPlugin(p input.BuiltinSourceInputMethod) error
	registerCredentialRepositoryPlugin(p credentialrepository.BuiltinCredentialRepositoryPlugin, consumerTypes []ocmruntime.Type) error
	registerSigningHandler(p signinghandler.BuiltinSigningHandler) error
	registerDigestProcessorPlugin(p digestprocessor.BuiltinDigestProcessorPlugin) error
}

type pluginsRegistrar struct{ pm *manager.PluginManager }

func (r *pluginsRegistrar) registerComponentVersionRepositoryPlugin(p componentversionrepository.BuiltinComponentVersionRepositoryProvider) error {
	return r.pm.ComponentVersionRepositoryRegistry.RegisterInternalComponentVersionRepositoryPlugin(p)
}

func (r *pluginsRegistrar) registerResourceInputPlugin(p input.BuiltinResourceInputMethod) error {
	return r.pm.InputRegistry.RegisterInternalResourceInputPlugin(p)
}

func (r *pluginsRegistrar) registerSourceInputPlugin(p input.BuiltinSourceInputMethod) error {
	return r.pm.InputRegistry.RegisterInternalSourceInputPlugin(p)
}

func (r *pluginsRegistrar) registerCredentialRepositoryPlugin(p credentialrepository.BuiltinCredentialRepositoryPlugin,
	consumerTypes []ocmruntime.Type) error {
	return r.pm.CredentialRepositoryRegistry.RegisterInternalCredentialRepositoryPlugin(p, consumerTypes)
}

func (r *pluginsRegistrar) registerSigningHandler(p signinghandler.BuiltinSigningHandler) error {
	return r.pm.SigningRegistry.RegisterInternalComponentSignatureHandler(p)
}

func (r *pluginsRegistrar) registerDigestProcessorPlugin(p digestprocessor.BuiltinDigestProcessorPlugin) error {
	return r.pm.DigestProcessorRegistry.RegisterInternalDigestProcessorPlugin(p)
}

func GetPluginManager(ctx context.Context, registry string) (*manager.PluginManager, error) {
	pm := manager.NewPluginManager(ctx)
	return setupPluginManager(ctx, pm, &pluginsRegistrar{pm}, registry)
}

func setupPluginManager(_ context.Context, pm *manager.PluginManager, r pluginManagerRegistrar,
	registry string) (*manager.PluginManager, error) {
	if err := r.registerComponentVersionRepositoryPlugin(
		provider.NewComponentVersionRepositoryProvider(provider.WithScheme(ocirepository.Scheme)),
	); err != nil {
		return nil, fmt.Errorf("failed to register internal component version repository plugin: %w", err)
	}

	if err := r.registerResourceInputPlugin(&file.InputMethod{WorkingDirectory: "./"}); err != nil {
		return nil, fmt.Errorf("failed to register file input plugin: %w", err)
	}

	if err := r.registerResourceInputPlugin(&dir.InputMethod{WorkingDirectory: "./"}); err != nil {
		return nil, fmt.Errorf("failed to register dir input plugin: %w", err)
	}

	if err := r.registerSourceInputPlugin(&file.InputMethod{WorkingDirectory: "./"}); err != nil {
		return nil, fmt.Errorf("failed to register file input plugin: %w", err)
	}

	if err := r.registerSourceInputPlugin(&dir.InputMethod{WorkingDirectory: "./"}); err != nil {
		return nil, fmt.Errorf("failed to register dir input plugin: %w", err)
	}

	if err := r.registerCredentialRepositoryPlugin(
		&ocicredentials.OCICredentialRepository{}, []ocmruntime.Type{v1identity.Type},
	); err != nil {
		return nil, fmt.Errorf("failed to register credential repository plugin: %w", err)
	}

	signingHandler, err := handler.New(signingv1alpha1.Scheme, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create signing handler: %w", err)
	}

	if err := r.registerSigningHandler(signingHandler); err != nil {
		return nil, fmt.Errorf("failed to register internal signing plugin: %w", err)
	}

	isHTTP := strings.HasPrefix(registry, "http://")
	if err := r.registerDigestProcessorPlugin(newPlainHTTPResourceRepository(isHTTP)); err != nil {
		return nil, fmt.Errorf("failed to register digest processor plugin: %w", err)
	}

	return pm, nil
}
