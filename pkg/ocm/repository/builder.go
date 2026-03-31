package repository

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	genericv1 "ocm.software/open-component-model/bindings/go/configuration/generic/v1/spec"
	"ocm.software/open-component-model/bindings/go/credentials"
	credentialsruntime "ocm.software/open-component-model/bindings/go/credentials/spec/config/runtime"
	credentialsv1 "ocm.software/open-component-model/bindings/go/credentials/spec/config/v1"
	ocicredentials "ocm.software/open-component-model/bindings/go/oci/credentials"
	"ocm.software/open-component-model/bindings/go/oci/repository/provider"
	"ocm.software/open-component-model/bindings/go/oci/repository/resource"
	ocispeccredentials "ocm.software/open-component-model/bindings/go/oci/spec/credentials"
	ociidentityv1 "ocm.software/open-component-model/bindings/go/oci/spec/credentials/identity/v1"
	ocicredentialsv1 "ocm.software/open-component-model/bindings/go/oci/spec/credentials/v1"
	"ocm.software/open-component-model/bindings/go/plugin/manager"
	"ocm.software/open-component-model/bindings/go/runtime"
	"ocm.software/open-component-model/bindings/go/transfer"
)

// OciClientBuilder constructs OciClient instances with a fluent, type-safe API.
//
// The builder pattern is the recommended approach for creating clients because it:
//   - Makes configuration explicit and self-documenting
//   - Handles the complexity of credential graph construction internally
//   - Provides sensible defaults while allowing customization
//   - Ensures all required dependencies are properly initialized
//
// A zero-value OciClientBuilder is ready to use and will create a client with:
//   - No authentication (suitable for public registries)
//   - Discarded logging (no output)
//   - Default OCI repository provider
//
// Use NewOciClientBuilder() for clarity, though it's functionally equivalent to &OciClientBuilder{}.
//
// Example usage:
//
//	client, err := NewOciClientBuilder().
//	    WithDockerConfigJsonSecret(pullSecret).
//	    WithLogger(ctrl.Log).
//	    Build(ctx)
//	if err != nil {
//	    return fmt.Errorf("building OCM client: %w", err)
//	}
type OciClientBuilder struct {
	log    logr.Logger
	secret *v1.Secret
}

// WithLogger configures structured logging for the OciClient.
//
// If not called, the client uses logr.Discard() which silently discards all log output.
//
// The logger should implement logr.Logger, which is the standard for Kubernetes ecosystem
// logging. Compatible implementations include:
//   - controller-runtime's logger (ctrl.Log)
//   - zapr (Zap adapter)
//   - logrr (logrus adapter)
//   - stdr (standard library log adapter)
//
// Example:
//
//	import ctrl "sigs.k8s.io/controller-runtime"
//
//	builder.WithLogger(ctrl.Log)
func (builder *OciClientBuilder) WithLogger(log logr.Logger) *OciClientBuilder {
	builder.log = log
	return builder
}

// WithDockerConfigJsonSecret configures authentication using a Kubernetes pull secret.
//
// The secret must be of type kubernetes.io/dockerconfigjson and contain a valid Docker
// config JSON in the .dockerconfigjson key.
//
// The Docker config JSON format is:
//
//	{
//	  "auths": {
//	    "registry.example.com": {
//	      "auth": "base64(username:password)"
//	    },
//	    "another-registry.io": {
//	      "username": "user",
//	      "password": "pass"
//	    }
//	  }
//	}
//
// During Build(), the secret is parsed and converted into an OCM credential graph that
// matches registry hostnames with credentials. The credential resolution is automatic
// during Get() and Save() operations.
//
// If a repository requires authentication but no matching credentials are found, the
// client will log a warning and attempt anonymous access, which succeeds for public
// repositories.
//
// Example Kubernetes Secret:
//
//	apiVersion: v1
//	kind: Secret
//	type: kubernetes.io/dockerconfigjson
//	metadata:
//	  name: ocm-registry-credentials
//	data:
//	  .dockerconfigjson: eyJhdXRocyI6eyJnaGNyLmlvIjp7ImF1dGgiOiJ1c2VyOnBhc3MifX19
//
// If not called, the client operates without authentication (suitable for public registries).
func (builder *OciClientBuilder) WithDockerConfigJsonSecret(secret *v1.Secret) *OciClientBuilder {
	builder.secret = secret
	return builder
}

// Build constructs and initializes an OciClient with the configured options.
//
// This method performs several initialization steps:
//  1. Creates a credential resolver from the provided secret (if any)
//  2. Initializes the OCI repository provider for component version access
//  3. Configures logging with appropriate defaults
//  4. Validates the configuration and returns any errors
//
// Credential Resolution:
//
// If a secret was provided via WithDockerConfigJsonSecret(), Build() will:
//   - Parse the Docker config JSON from the secret's .dockerconfigjson key
//   - Convert it to OCM's internal credential format
//   - Build a credential graph that resolves registry hostnames to credentials
//   - Register the OCI credential repository plugin for runtime resolution
//
// If no secret was provided, a NoopCredentialResolver is used, which always returns
// no credentials. This works fine for public registries but will fail authentication
// for private registries.
//
// Example:
//
//	builder := repository.NewOciClientBuilder().
//	    WithDockerConfigJsonSecret(secret).
//	    WithLogger(logger)
//
//	client, err := builder.Build(ctx)
//	if err != nil {
//	    return fmt.Errorf("failed to build OCM client: %w", err)
//	}
//	// client is ready for Get() and Save() operations
func (builder *OciClientBuilder) Build(ctx context.Context) (Client, error) {
	var (
		resolver         credentials.Resolver = NoopCredentialResolver{}
		repoProvider                          = provider.NewComponentVersionRepositoryProvider()
		transferExecutor TransferExecutor
		opts             []OciClientOption
	)
	if builder.secret != nil {
		cfg, err := builder.getGenericConfigurationFromSecret()
		if err != nil {
			return nil, fmt.Errorf("getting generic configuration from secret: %w", err)
		}
		if cfg == nil {
			return nil, fmt.Errorf("no credential configuration could be derived from secret")
		}
		resolver, err = buildGraph(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("building credential graph from config: %w", err)
		}
	} else {
		if !builder.log.IsZero() {
			builder.log.Info("no secret provided for OCI client builder, using noop credential resolver")
		}
	}
	transferExecutor = NewDefaultTransferExecutor(
		transfer.NewDefaultBuilder(repoProvider, resource.NewResourceRepository(nil), resolver),
	)
	if !builder.log.IsZero() {
		opts = append(opts, WithOciClientLogger(builder.log))
	}
	return NewOciClient(resolver, repoProvider, transferExecutor, opts...), nil
}

func (builder *OciClientBuilder) getGenericConfigurationFromSecret() (*credentialsruntime.Config, error) {
	dockerConfigJson, ok := builder.secret.Data[v1.DockerConfigJsonKey]
	if !ok {
		return nil, fmt.Errorf("secret does not contain key %q", v1.DockerConfigJsonKey)
	}

	dockerConfig := &ocicredentialsv1.DockerConfig{}
	if _, err := ocispeccredentials.Scheme.DefaultType(dockerConfig); err != nil {
		return nil, fmt.Errorf("failed to get default type for docker config type: %w", err)
	}

	dockerConfig.DockerConfig = string(dockerConfigJson)
	// Validate that dockerConfigJson is valid Docker config JSON
	raw := &runtime.Raw{}
	if err := ocispeccredentials.Scheme.Convert(dockerConfig, raw); err != nil {
		return nil, fmt.Errorf("failed to convert docker config to raw: %w", err)
	}

	credScheme := runtime.NewScheme()
	credentialsv1.MustRegister(credScheme)
	credConfig := &credentialsv1.Config{
		Repositories: []credentialsv1.RepositoryConfigEntry{{Repository: raw}},
	}

	rawCreds := &runtime.Raw{}
	if err := credScheme.Convert(credConfig, rawCreds); err != nil {
		return nil, fmt.Errorf("failed to convert credential config to raw type: %w", err)
	}

	cfg := &genericv1.Config{
		Type:           runtime.Type{Version: genericv1.Version, Name: genericv1.ConfigType},
		Configurations: []*runtime.Raw{rawCreds},
	}
	credCfg, err := credentialsruntime.LookupCredentialConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup credential config from generic config: %w", err)
	}
	return credCfg, nil
}

func buildGraph(ctx context.Context, cfg *credentialsruntime.Config) (credentials.Resolver, error) {
	pm := manager.NewPluginManager(ctx)
	if err := pm.CredentialRepositoryRegistry.RegisterInternalCredentialRepositoryPlugin(
		&ocicredentials.OCICredentialRepository{}, []runtime.Type{ociidentityv1.Type}); err != nil {
		return nil, fmt.Errorf("failed to register OCI credential repository plugin: %w", err)
	}
	var (
		graph credentials.Resolver
		err   error
	)
	graph, err = credentials.ToGraph(ctx, cfg, credentials.Options{
		CredentialRepositoryTypeScheme: pm.CredentialRepositoryRegistry.RepositoryScheme(),
		RepositoryPluginProvider:       pm.CredentialRepositoryRegistry,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create credential graph from config: %w", err)
	}
	return graph, nil
}

// NewOciClientBuilder creates a new OciClientBuilder with default settings.
//
// The returned builder is ready to use and will create a client with:
//   - No authentication (anonymous access)
//   - Discarded logging (no output)
//   - Standard OCI repository provider
//
// Chain configuration methods before calling Build():
//
//	client, err := NewOciClientBuilder().
//	    WithDockerConfigJsonSecret(secret).
//	    WithLogger(logger).
//	    Build(ctx)
func NewOciClientBuilder() *OciClientBuilder {
	return &OciClientBuilder{}
}
