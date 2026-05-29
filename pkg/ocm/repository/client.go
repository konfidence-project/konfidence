package repository

//go:generate go run go.uber.org/mock/mockgen -destination=internal/mocks/mock_client_port.go -package=mocks github.com/konfidence-project/konfidence/pkg/ocm/repository Client

import (
	"context"
	"errors"

	konfcompref "github.com/konfidence-project/konfidence/pkg/ocm/compref"
	"ocm.software/open-component-model/bindings/go/blob"
	descruntime "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	"ocm.software/open-component-model/bindings/go/repository"
	"ocm.software/open-component-model/bindings/go/runtime"
)

var (
	// ErrComponentAlreadyExists is returned when attempting to save a component descriptor
	// that already exists in the repository with the same name and version.
	ErrComponentAlreadyExists = errors.New("component already exists")

	// ErrNotFound is returned when a component descriptor or component version cannot be
	// found in the repository.
	ErrNotFound = repository.ErrNotFound

	// ErrInvalidComponentReference is returned when a component reference is invalid,
	// such as when the version field is empty. All repository operations require an
	// explicit version (semantic version or alias).
	ErrInvalidComponentReference = konfcompref.ErrInvalidComponentReference
)

// ResourceReadClient defines read operations for accessing local resources stored in
// OCM component versions.
type ResourceReadClient interface {
	// GetLocalResource retrieves the content and metadata of a local resource identified
	// by its component reference and resource identity.
	//
	// The identity parameter is a map of extra identity attributes that uniquely identify
	// the resource within the component version (e.g. {"name": "my-resource"}).
	//
	// Returns the raw blob content as a blob.ReadOnlyBlob (call ReadCloser() and io.ReadAll to
	// get the bytes) along with the resource metadata. Returns ErrNotFound if no matching
	// resource exists.
	//
	// Example:
	//
	//	identity := runtime.Identity{"name": "my-manifest"}
	//	b, res, err := client.GetLocalResource(ctx, ref, identity)
	//	if err != nil {
	//	    return err
	//	}
	//	rc, err := b.ReadCloser()
	//	if err != nil {
	//	    return err
	//	}
	//	defer rc.Close()
	//	data, err := io.ReadAll(rc)
	GetLocalResource(
		ctx context.Context,
		ref compref.Ref,
		identity runtime.Identity) (blob.ReadOnlyBlob, *descruntime.Resource, error)
}

// ReadClient defines read operations on OCM component versions.
type ReadClient interface {
	// Get retrieves a component descriptor by reference.
	//
	// The ref.Version field can specify either:
	//   - A semantic version (e.g., "1.2.3")
	//   - A version alias (e.g., "latest", "stable", "production")
	//
	// Returns ErrInvalidComponentReference if the reference is incomplete or invalid.
	//
	// Returns ErrNotFound if:
	//   - The component doesn't exist in the repository
	//   - The specified version or alias doesn't exist
	//   - The repository doesn't exist or is inaccessible
	//
	// Example:
	//
	//	// Get a specific semantic version
	//	ref := compref.Ref{
	//	    Repository: ociSpec,
	//	    Component:  "github.com/acme/backend",
	//	    Version:    "1.2.3",
	//	}
	//	desc, err := client.Get(ctx, ref)
	//
	//	// Get using a version alias
	//	ref.Version = "latest"
	//	desc, err = client.Get(ctx, ref)
	Get(ctx context.Context, ref compref.Ref) (descruntime.Descriptor, error)

	ResourceReadClient
}

// WriteClient defines write operations for persisting component descriptors to OCM repositories.
type WriteClient interface {
	// Save persists a component descriptor to the specified repository.
	//
	// The descriptor's component name and version are used as the identity. Once saved,
	// component descriptors are immutable - attempting to save the same component and
	// version again returns ErrComponentAlreadyExists.
	//
	// The repoSpec parameter defines the target repository. For OCI repositories, it
	// typically contains the registry hostname and repository path.
	//
	// Returns ErrComponentAlreadyExists if an identical component version already exists.
	// This is not considered an error condition for idempotent operations.
	//
	// Example:
	//
	//	repoSpec := runtime.Typed{
	//	    Type: "oci",
	//	    Data: map[string]interface{}{
	//	        "baseUrl": "ghcr.io/acme/components",
	//	    },
	//	}
	//
	//	err := client.Save(ctx, repoSpec, descriptor)
	//	if errors.Is(err, repository.ErrComponentAlreadyExists) {
	//	    // Already published - safe to continue
	//	}
	Save(ctx context.Context, repoSpec runtime.Typed, descriptor descruntime.Descriptor) error

	// Copy transfers references to the specified target repository.
	//
	// The targetRepoSpec parameter defines the target repository. For OCI repositories, it
	// typically contains the registry hostname and repository path.
	//
	// Returns ErrInvalidComponentReference if any reference is incomplete or invalid.
	//
	// Example:
	//
	//	targetRepoSpec := runtime.Typed{
	//	    Type: "oci",
	//	    Data: map[string]interface{}{
	//	        "baseUrl": "ghcr.io/acme/components",
	//	    },
	//	}
	//
	// err := client.Copy(ctx, artifactReferences, targetRepoSpec)
	Copy(ctx context.Context, artifactReferences []compref.Ref, targetRepoSpec runtime.Typed) error

	// AddAlias creates or updates a version alias for a component.
	//
	// Version aliases provide human-readable names (like "latest", "stable", "production")
	// that point to specific component versions. Aliases are mutable and can be updated
	// to point to different versions over time.
	//
	// The ref parameter must specify:
	//   - Repository: the target OCI repository
	//   - Component: the component name
	//   - Version: the version to alias (e.g., "1.2.3", or an existing alias "edge")
	//
	// The alias parameter is the human-readable name to assign (e.g., "latest", "stable").
	//
	// If the alias already exists, it will be updated to point to the new version.
	//
	// Returns ErrInvalidComponentReference if the reference or alias is incomplete or invalid.
	// Returns ErrNotFound if the specified component version doesn't exist.
	//
	// Example:
	//
	//	// Point the "latest" alias to version 2.1.0
	//	ref := compref.Ref{
	//	    Repository: ociSpec,
	//	    Component:  "github.com/acme/backend",
	//	    Version:    "2.1.0",
	//	}
	//	err := client.AddAlias(ctx, ref, "latest")
	//
	//	// Users can now fetch using the alias
	//	aliasRef := compref.Ref{
	//	    Repository: ociSpec,
	//	    Component:  "github.com/acme/backend",
	//	    Version:    "latest",
	//	}
	//	desc, err := client.Get(ctx, aliasRef)
	AddAlias(ctx context.Context, ref compref.Ref, alias string) error
}

// Client combines read and write access to component descriptors in OCM repositories.
//
// The primary implementation is OciClient, which works with OCI-compliant registries.
type Client interface {
	ReadClient
	WriteClient
}
