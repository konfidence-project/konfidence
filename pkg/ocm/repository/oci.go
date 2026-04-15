package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	konfcompref "github.com/konfidence-project/pkg/ocm/compref"
	"ocm.software/open-component-model/bindings/go/credentials"
	descruntime "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	"ocm.software/open-component-model/bindings/go/oci"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	"ocm.software/open-component-model/bindings/go/repository"
	"ocm.software/open-component-model/bindings/go/runtime"
	"ocm.software/open-component-model/bindings/go/transfer"
)

var (
	_ Client = (*OciClient)(nil)
)

// OciClient provides read and write access to component descriptors stored in OCI registries.
//
// # Authentication
//
// Authentication is handled through the credentials.Resolver provided during construction.
// The resolver is queried for each repository access using the repository's credential
// consumer identity. If no credentials are found, the client attempts anonymous access.
//
// # Construction
//
// Use OciClientBuilder for standard use cases:
//
//	client, err := NewOciClientBuilder().
//	    WithDockerConfigJsonSecret(secret).
//	    WithLogger(logger).
//	    Build(ctx)
//
// Or use NewOciClient directly when you need custom credential resolution:
//
//	client := NewOciClient(customResolver, provider, transferExecutor, WithOciClientLogger(logger))
type OciClient struct {
	log              logr.Logger
	resolver         credentials.Resolver
	provider         repository.ComponentVersionRepositoryProvider
	transferExecutor TransferExecutor
}

// Get retrieves a component descriptor from an OCI registry by reference.
//
// # Error Handling
//
// Returns ErrInvalidComponentReference if the reference is incomplete or invalid.
// Returns ErrNotFound if the component version doesn't exist in the repository.
// Returns a generic error in case of other failures (e.g., repository access issues).
func (c OciClient) Get(ctx context.Context, ref compref.Ref) (descruntime.Descriptor, error) {
	if err := konfcompref.Validate(ref); err != nil {
		return descruntime.Descriptor{},
			errors.Join(ErrInvalidComponentReference, fmt.Errorf("invalid reference for Get: %w", err))
	}
	repo, err := c.getRepo(ctx, ref.Repository)
	if err != nil {
		return descruntime.Descriptor{},
			fmt.Errorf("getting repository for component reference %s: %w", ref, err)
	}
	desc, err := repo.GetComponentVersion(ctx, ref.Component, ref.Version)
	if errors.Is(err, repository.ErrNotFound) {
		return descruntime.Descriptor{},
			errors.Join(err, fmt.Errorf("%s: %w, target version %s", ref, ErrNotFound, ref.Version))
	} else if err != nil {
		return descruntime.Descriptor{},
			fmt.Errorf("getting component version %s for component %s: %w", ref.Version, ref, err)
	}
	return *desc, nil
}

func (c OciClient) getRepo(ctx context.Context, repoSpec runtime.Typed) (oci.ComponentVersionRepository, error) {
	identity, err := c.provider.GetComponentVersionRepositoryCredentialConsumerIdentity(ctx, repoSpec)
	if err != nil {
		return nil, fmt.Errorf("getting credential consumer identity for %s: %w", repoSpec, err)
	}
	creds, err := c.resolver.Resolve(ctx, identity)
	if errors.Is(err, credentials.ErrNotFound) {
		c.log.Info("no credentials found for repository, trying to access repository without credentials",
			"repository", repoSpec, "identity", identity)
	} else if err != nil {
		return nil, fmt.Errorf("resolving credentials for identity %s: %w", identity, err)
	}
	repo, err := c.provider.GetComponentVersionRepository(ctx, repoSpec, creds)
	if err != nil {
		return nil, err
	}
	ociRepo, ok := repo.(oci.ComponentVersionRepository)
	if !ok {
		return nil, fmt.Errorf(
			"expected an OCI component version repository for spec %s, but got %T. (targeting unsupported CTF?)",
			repoSpec, repo,
		)
	}
	return ociRepo, nil
}

// Save persists a component descriptor to an OCI registry.
//
// # Immutability Check
//
// Before saving, Save checks if the component version already exists. If it does,
// Save returns ErrComponentAlreadyExists without modifying the repository.
func (c OciClient) Save(ctx context.Context, repoSpec runtime.Typed, desc descruntime.Descriptor) error {
	repo, err := c.getRepo(ctx, repoSpec)
	if err != nil {
		return fmt.Errorf("getting repository %s for component %s: %w",
			repoSpec, desc.Component.ToIdentity(), err)
	}
	_, err = repo.GetComponentVersion(ctx, desc.Component.Name, desc.Component.Version)
	if errors.Is(err, repository.ErrNotFound) {
		if err = repo.AddComponentVersion(ctx, &desc); err != nil {
			return fmt.Errorf("add component %s to repository %s: %w",
				desc.Component.ToIdentity(), repoSpec, err)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("check if component version already exists: %w", err)
	}
	return fmt.Errorf("%w: component version %s already exists in repository %s",
		ErrComponentAlreadyExists, desc.Component.ToIdentity(), repoSpec)
}

// Copy transfers references to the specified target repository.
//
// Returns ErrInvalidComponentReference if any reference is incomplete or invalid.
func (c OciClient) Copy(ctx context.Context, artifactReferences []compref.Ref, targetRepoSpec runtime.Typed) error {
	transferOptions := make([]transfer.Option, 0, len(artifactReferences)+1)
	for _, ref := range artifactReferences {
		if err := konfcompref.Validate(ref); err != nil {
			return errors.Join(ErrInvalidComponentReference, fmt.Errorf("invalid reference for Copy: %w", err))
		}
		repo, err := c.getRepo(ctx, ref.Repository)
		if err != nil {
			return fmt.Errorf("getting repository for component reference %s: %w", ref, err)
		}
		transferOptions = append(transferOptions, transfer.WithTransfer(
			transfer.Component(ref.Component, ref.Version),
			transfer.FromRepository(repo, ref.Repository),
			transfer.ToRepositorySpec(targetRepoSpec),
		))
	}
	transferOptions = append(transferOptions, transfer.WithRecursive(true))
	transferGraphDefinition, err := transfer.BuildGraphDefinition(ctx, transferOptions...)
	if err != nil {
		return fmt.Errorf("building transfer graph definition: %w", err)
	}
	if err := c.transferExecutor.Execute(ctx, transferGraphDefinition); err != nil {
		return fmt.Errorf("executing copy: %w", err)
	}
	return nil
}

// AddAlias creates or updates a version alias for a component in the OCI registry.
//
// This implementation delegates to the underlying OCI repository's AddComponentVersionAlias
// method, which creates an OCI tag (alias) pointing to the manifest/index of the specified
// component version.
//
// The ref.Version can be an existing exact semantic version (e.g., "1.2.3"), or an existing alias.
// The alias parameter is the tag name to create (e.g., "latest", "stable").
//
// If the alias already exists, it will be overwritten to point to the new version.
// This enables updating rolling aliases like "latest".
//
// Example:
//
//	ref := compref.Ref{
//	    Repository: ociSpec,
//	    Component:  "github.com/acme/backend",
//	    Version:    "2.1.0",
//	}
//	err := client.AddAlias(ctx, ref, "latest")
//
// # Error Handling
//
// Returns ErrInvalidComponentReference if the reference is incomplete or invalid.
// Returns ErrNotFound if the component version doesn't exist in the repository.
// Returns a generic error in case of other failures (e.g., repository access issues).
func (c OciClient) AddAlias(ctx context.Context, ref compref.Ref, alias string) error {
	if err := konfcompref.Validate(ref); err != nil {
		return errors.Join(ErrInvalidComponentReference, fmt.Errorf("invalid reference for AddAlias: %w", err))
	}
	if alias == "" {
		return errors.New("invalid input for AddAlias: %w: alias must not be empty")
	}
	repo, err := c.getRepo(ctx, ref.Repository)
	if err != nil {
		return fmt.Errorf("getting repository for component reference %s: %w", ref, err)
	}
	err = repo.AddComponentVersionAlias(ctx, ref.Component, ref.Version, alias)
	if errors.Is(err, repository.ErrNotFound) {
		return errors.Join(err, fmt.Errorf("%s: %w", ref, ErrNotFound))
	} else if err != nil {
		return fmt.Errorf("adding alias %s to component version %s: %w", alias, ref, err)
	}
	return nil
}

// OciClientOption configures optional behavior for an OciClient.
//
// Use the provided With* functions to create options:
//
//	client := NewOciClient(resolver, provider,
//	    WithOciClientLogger(logger),
//	)
type OciClientOption func(*OciClient)

// WithOciClientLogger configures structured logging for an OciClient.
//
// If this option is not provided, the client uses logr.Discard() which silently
// discards all log output.
func WithOciClientLogger(log logr.Logger) OciClientOption {
	return func(c *OciClient) {
		c.log = log.WithName("ocm-oci-client")
	}
}

// NewOciClient creates a new OciClient with the specified credential resolver and
// repository provider.
//
// # Parameters
//
// resolver: Resolves credentials for OCI registries. Query is by credential consumer
// identity (typically the registry hostname).
//
// provider: Provides access to OCI repositories and component version operations. The
// standard implementation is provider.NewComponentVersionRepositoryProvider().
//
// transferExecutor: Executes component transfers for Copy operations.
//
// options: Optional configuration. Use WithOciClientLogger to enable logging.
//
// # Construction Pattern
//
// For most use cases, use OciClientBuilder instead of calling NewOciClient directly:
//
//	// Recommended: use builder
//	client, err := NewOciClientBuilder().
//	    WithDockerConfigJsonSecret(secret).
//	    Build(ctx)
//
//	// Advanced: construct directly
//	client := NewOciClient(
//	    customResolver,
//	    provider.NewComponentVersionRepositoryProvider(),
//	    transferExecutor,
//	    WithOciClientLogger(logger),
//	)
func NewOciClient(
	resolver credentials.Resolver,
	provider repository.ComponentVersionRepositoryProvider,
	transferExecutor TransferExecutor,
	options ...OciClientOption,
) OciClient {
	client := OciClient{provider: provider, resolver: resolver, transferExecutor: transferExecutor}
	for _, opt := range options {
		opt(&client)
	}
	applyDefaults(&client)
	return client
}

func applyDefaults(c *OciClient) {
	if c.log.IsZero() {
		c.log = logr.Discard()
	}
}
