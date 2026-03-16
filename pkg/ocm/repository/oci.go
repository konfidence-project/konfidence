package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"ocm.software/open-component-model/bindings/go/credentials"
	descruntime "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	"ocm.software/open-component-model/bindings/go/repository"
	"ocm.software/open-component-model/bindings/go/runtime"
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
//	client := NewOciClient(customResolver, provider, WithOciClientLogger(logger))
type OciClient struct {
	log      logr.Logger
	resolver credentials.Resolver
	provider repository.ComponentVersionRepositoryProvider
}

// Get retrieves a component descriptor from an OCI registry by reference.
//
// # Version Resolution
//
// If ref.Version is empty, Get fetches the latest version by:
//  1. Listing all versions of the component
//  2. Sorting by semantic version
//  3. Returning the highest version
//
// If ref.Version is specified, Get fetches that exact version.
//
// # Error Handling
//
// Returns ErrNotFound if:
//   - The component doesn't exist in the repository
//   - The specified version doesn't exist
//   - The repository doesn't exist (404 from registry)
//   - No versions exist for the component when fetching latest
func (c OciClient) Get(ctx context.Context, ref compref.Ref) (descruntime.Descriptor, error) {
	repo, err := c.getRepo(ctx, ref.Repository)
	if err != nil {
		return descruntime.Descriptor{},
			fmt.Errorf("getting repository for component reference %s: %w", ref, err)
	}
	version := ref.Version
	if version == "" {
		var err error
		version, err = c.getLatestComponentVersion(ctx, repo, ref)
		if err != nil {
			return descruntime.Descriptor{},
				fmt.Errorf("getting latest component version: %w", err)
		}
	}
	desc, err := repo.GetComponentVersion(ctx, ref.Component, version)
	if errors.Is(err, repository.ErrNotFound) {
		return descruntime.Descriptor{},
			fmt.Errorf("%s: %w, target version %s", ref, ErrNotFound, version)
	} else if err != nil {
		return descruntime.Descriptor{},
			fmt.Errorf("getting component version %s for component %s: %w", version, ref, err)
	}
	return *desc, nil
}

// GetLocalResource retrieves the content and metadata of a locally-stored resource from an
// OCI registry by component reference and resource identity.
//
// This is used for resources whose content is embedded as OCI layers in the component version
// (e.g. resources with access type "localBlob"), as opposed to externally-referenced resources
// like helmChart or ociImage whose access specs are merely pointers to external registries.
func (c OciClient) GetLocalResource(
	ctx context.Context,
	ref compref.Ref,
	identity runtime.Identity,
) (ReadOnlyBlob, *Resource, error) {
	repo, err := c.getRepo(ctx, ref.Repository)
	if err != nil {
		return nil, nil, fmt.Errorf("getting repository for component reference %s: %w", ref, err)
	}
	content, resource, err := repo.GetLocalResource(ctx, ref.Component, ref.Version, identity)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, nil, fmt.Errorf("%s identity %v: %w", ref, identity, ErrNotFound)
	} else if err != nil {
		return nil, nil, fmt.Errorf("getting local resource for %s identity %v: %w", ref, identity, err)
	}
	return content, resource, nil
}

func (c OciClient) getRepo(ctx context.Context, repoSpec runtime.Typed) (repository.ComponentVersionRepository, error) {
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
	return c.provider.GetComponentVersionRepository(ctx, repoSpec, creds)
}

func (c OciClient) getLatestComponentVersion(
	ctx context.Context, repo repository.ComponentVersionRepository, ref compref.Ref) (string, error) {
	versions, err := repo.ListComponentVersions(ctx, ref.Component)
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "repository name not known to registry") {
		return "", fmt.Errorf("no versions found for component %s: %w", ref, ErrNotFound)
	} else if err != nil {
		return "", fmt.Errorf("list component versions for component %s: %w", ref, err)
	}
	if len(versions) == 0 {
		return "", fmt.Errorf("no versions found for component %s: %w", ref, ErrNotFound)
	}
	return versions[0], nil
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
//	    WithOciClientLogger(logger),
//	)
func NewOciClient(
	resolver credentials.Resolver,
	provider repository.ComponentVersionRepositoryProvider,
	options ...OciClientOption,
) OciClient {
	client := OciClient{provider: provider, resolver: resolver}
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
