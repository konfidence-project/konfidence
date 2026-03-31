package repository

import (
	"context"
	"errors"

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
)

// ReadClient defines read-only operations for accessing component descriptors stored in
// OCM repositories.
//
// Implementations must be safe for concurrent use by multiple goroutines.
type ReadClient interface {
	// Get retrieves a component descriptor by reference.
	//
	// If ref.Version is empty, Get returns the latest version of the component based on
	// semantic versioning rules. The repository determines version ordering.
	//
	// Returns ErrNotFound if:
	//   - The component doesn't exist in the repository
	//   - The specified version doesn't exist
	//   - The repository doesn't exist or is inaccessible
	//
	// Example:
	//
	//	// Get a specific version
	//	ref := compref.Ref{
	//	    Repository: ociSpec,
	//	    Component:  "github.com/acme/backend",
	//	    Version:    "1.2.3",
	//	}
	//	desc, err := client.Get(ctx, ref)
	//
	//	// Get the latest version
	//	ref.Version = ""
	//	latest, err := client.Get(ctx, ref)
	Get(ctx context.Context, ref compref.Ref) (descruntime.Descriptor, error)
}

// WriteClient defines write operations for persisting component descriptors to OCM repositories.
//
// Implementations must be safe for concurrent use by multiple goroutines.
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
}

// Client combines read and write access to component descriptors in OCM repositories.
//
// Implementations must be safe for concurrent use by multiple goroutines.
//
// The primary implementation is OciClient, which works with OCI-compliant registries.
type Client interface {
	ReadClient
	WriteClient
}
