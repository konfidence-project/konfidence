package ocm

import (
	"context"
	"fmt"
	"os"

	"github.com/konfidence-project/konfidence/internal/kden/log"
	"ocm.software/open-component-model/bindings/go/constructor"
	constructorruntime "ocm.software/open-component-model/bindings/go/constructor/runtime"

	"github.com/spf13/cobra"
	ocmgenericspecv1 "ocm.software/open-component-model/bindings/go/configuration/generic/v1/spec"
	ocmconstructorspecv1 "ocm.software/open-component-model/bindings/go/constructor/spec/v1"
	"ocm.software/open-component-model/bindings/go/credentials"
	credentialsruntime "ocm.software/open-component-model/bindings/go/credentials/spec/config/runtime"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	ocmocirepository "ocm.software/open-component-model/bindings/go/oci/spec/repository"
	ocmocispecctfv1 "ocm.software/open-component-model/bindings/go/oci/spec/repository/v1/ctf"
	"ocm.software/open-component-model/bindings/go/plugin/manager"
	ocmrepository "ocm.software/open-component-model/bindings/go/repository"
	"ocm.software/open-component-model/bindings/go/repository/component/resolvers"
	"ocm.software/open-component-model/bindings/go/runtime"
	"ocm.software/open-component-model/cli/cmd/configuration"
)

type RepositoryResolverOptions struct {
	config            *ocmgenericspecv1.Config
	repository        runtime.Typed
	componentPatterns []string
}

type RepositoryResolverOption func(*RepositoryResolverOptions)

func GetRepositorySpec(repository string) (runtime.Typed, error) {
	typed, err := compref.ParseRepository(repository)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository: %w", err)
	}

	if ctfRepo, ok := typed.(*ocmocispecctfv1.Repository); ok {
		var accessMode ocmocispecctfv1.AccessMode = ocmocispecctfv1.AccessModeReadWrite
		if _, err := os.Stat(ctfRepo.FilePath); os.IsNotExist(err) {
			accessMode += "|" + ocmocispecctfv1.AccessModeCreate
		}

		log.Debug("setting access mode for CTF repository", "path", ctfRepo.FilePath, "ref", repository, "mode", accessMode)
		ctfRepo.AccessMode = accessMode
	}

	return typed, nil
}

func GetComponentRepositoryResolver(
	ctx context.Context,
	repoProvider ocmrepository.ComponentVersionRepositoryProvider,
	credentialGraph credentials.Resolver,
	opts ...RepositoryResolverOption) (resolvers.ComponentVersionRepositoryResolver, error) {
	options := &RepositoryResolverOptions{}
	for _, opt := range opts {
		opt(options)
	}

	fallbackResolvers, pathMatchers, err := resolvers.ExtractResolvers(options.config, ocmocirepository.Scheme)
	if err != nil {
		return nil, err
	}

	providerOpts := resolvers.Options{
		RepoProvider:      repoProvider,
		CredentialGraph:   credentialGraph,
		PathMatchers:      pathMatchers,
		FallbackResolvers: fallbackResolvers,
		ComponentPatterns: options.componentPatterns,
	}

	return resolvers.New(ctx, providerOpts, options.repository)

}

func WithConfig(config *ocmgenericspecv1.Config) RepositoryResolverOption {
	return func(o *RepositoryResolverOptions) {
		o.config = config
	}
}

func WithRepository(repo runtime.Typed) RepositoryResolverOption {
	return func(o *RepositoryResolverOptions) {
		o.repository = repo
	}
}

func WithComponentRef(ref *compref.Ref) RepositoryResolverOption {
	return func(o *RepositoryResolverOptions) {
		if ref == nil {
			return
		}
		o.repository = ref.Repository
		if ref.Component != "" {
			o.componentPatterns = []string{ref.Component}
		}
	}
}

func GetCredentialGraph(ctx context.Context, pluginManager *manager.PluginManager, config *ocmgenericspecv1.Config) (credentials.Resolver, error) {
	opts := credentials.Options{
		RepositoryPluginProvider: pluginManager.CredentialRepositoryRegistry,
		CredentialPluginProvider: credentials.GetCredentialPluginFn(
			func(ctx context.Context, typed runtime.Typed) (credentials.CredentialPlugin, error) {
				return nil, fmt.Errorf("no credential plugin found for type %s", typed)
			},
		),
		CredentialRepositoryTypeScheme: pluginManager.CredentialRepositoryRegistry.RepositoryScheme(),
	}
	credCfg, err := credentialsruntime.LookupCredentialConfig(config)
	if err != nil {
		return nil, err
	}
	if credCfg == nil {
		// No usable credentials — either no .ocmconfig at all, or one without a
		// credentials section. Registries are then accessed unauthenticated;
		// auth failures surface later as a 401/403. Tell the user so
		// unauthenticated access is visible, not silent. Also, ToGraph nil-derefs
		// on nil, so hand it an empty config.
		log.Info("no OCM credentials configured; registries will be accessed without credentials (unauthenticated)")
		credCfg = &credentialsruntime.Config{}
	}
	return credentials.ToGraph(ctx, credCfg, opts)
}

func ReadConstructorFromFile(filePath string) (*ocmconstructorspecv1.ComponentConstructor, error) {
	descriptorBytes, err := os.ReadFile(filePath)
	if err != nil {
		readError := fmt.Errorf("failed to read constructor file: %v", err)
		log.Error(err.Error())
		return nil, readError
	}

	componentConstructor, err := ParseComponentConstructor(string(descriptorBytes), filePath)
	if err != nil {
		unmarshallError := fmt.Errorf("failed to unmarshall constructor: %v", err)
		log.Error(unmarshallError.Error())
		return nil, unmarshallError
	}
	return componentConstructor, nil

}

func GetOcmConfiguration(cmd *cobra.Command) (*ocmgenericspecv1.Config, error) {
	cfg, err := configuration.GetFlattenedOCMConfigForCommand(cmd)
	if err != nil {
		// GetFlattenedOCMConfigForCommand only returns a nil config together
		// with an error (a missing .ocmconfig is the common case); on success it
		// is always non-nil. So this is the single place cfg can be nil.
		// A missing config is legitimate — kden then runs with defaults /
		// unauthenticated (the credential-specific notice is emitted in
		// GetCredentialGraph). Materialize an empty config so nil never reaches
		// downstream consumers (plugin manager, repository resolver, credential graph).
		log.Info("no OCM configuration loaded; proceeding with defaults", "error", err)
		cfg = &ocmgenericspecv1.Config{}
	}
	return cfg, nil
}

func NewComponentRepositoryResolver(
	ctx context.Context,
	repoProvider ocmrepository.ComponentVersionRepositoryProvider,
	credentialGraph credentials.Resolver,
	opts ...RepositoryResolverOption,
) (resolvers.ComponentVersionRepositoryResolver, error) {
	options := &RepositoryResolverOptions{}
	for _, opt := range opts {
		opt(options)
	}

	fallbackResolvers, pathMatchers, err := resolvers.ExtractResolvers(options.config, ocmocirepository.Scheme)
	if err != nil {
		return nil, err
	}

	providerOpts := resolvers.Options{
		RepoProvider:      repoProvider,
		CredentialGraph:   credentialGraph,
		PathMatchers:      pathMatchers,
		FallbackResolvers: fallbackResolvers,
		ComponentPatterns: options.componentPatterns,
	}

	return resolvers.New(ctx, providerOpts, options.repository)
}

func GetTargetRepository(
	ctx context.Context,
	constructorRuntime *constructorruntime.Component,
	provider *ConstructorProvider,
) (constructor.TargetRepository, error) {
	return provider.GetTargetRepository(ctx, constructorRuntime)
}
