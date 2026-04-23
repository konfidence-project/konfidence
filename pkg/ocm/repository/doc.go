// Package repository provides a Go client for reading and writing component descriptors
// in Open Component Model (OCM) repositories.
//
// A component descriptor is the core element of OCM that describes:
//   - The component's identity (name, version)
//   - Resources (deployable artifacts like container images, helm charts)
//   - Sources (source code references)
//   - References to other components (dependencies)
//   - Signatures for verification
//
// # Getting Started with the Builder
//
// The recommended way to create a client is using the builder pattern with OciClientBuilder.
// This provides a fluent API for configuring authentication, logging, and other options:
//
//	import (
//	    "context"
//	    "log"
//
//	    corev1 "k8s.io/api/core/v1"
//	    "github.com/go-logr/logr"
//	    "ocm.software/open-component-model/bindings/go/oci/compref"
//	    "path/to/repository"
//	)
//
//	func main() {
//	    ctx := context.Background()
//
//	    // Create a client with no authentication (for public registries)
//	    client, err := repository.NewOciClientBuilder().
//	        WithLogger(logr.Discard()).
//	        Build(ctx)
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//
//	    // Retrieve a component descriptor
//	    ref := compref.Ref{
//	        Repository: repository.MustParseOCIRef("ghcr.io/my-org/components"),
//	        Component:  "github.com/my-org/my-component",
//	        Version:    "1.0.0",
//	    }
//
//	    descriptor, err := client.Get(ctx, ref)
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//
//	    // Use the descriptor...
//	}
//
// # Client Provider
//
// Use ClientProvider to obtain an authenticated Client without managing credential resolution directly.
// It handles credential resolution, including loading OCM configurations from the
// Kubernetes API, so controllers do not need to manage this directly.
//
// Use DefaultOciClientProvider for standard use cases:
//
//	client, err := repository.DefaultOciClientProvider.NewClient(
//	    ctx,
//	    r.Client,          // sigs.k8s.io/controller-runtime/pkg/client.Reader
//	    req.Namespace,
//	    credentialsConfig, // []global.CredentialsConfig from the reconciled resource
//	)
//	if err != nil {
//	    return ctrl.Result{}, err
//	}
//
// For testing, use ClientProviderFunc to adapt any function to the ClientProvider interface:
//
//	fakeProvider := repository.ClientProviderFunc(
//	    func(ctx context.Context, k8sClient client.Reader, namespace string, creds []global.CredentialsConfig) (repository.Client, error) {
//	        return myFakeClient, nil
//	    },
//	)
//
// # Reading Component Descriptors
//
// Use the Get method to retrieve component descriptors by reference.
// You must specify an explicit version (semantic version or alias):
//
//	// Get a specific semantic version
//	ref := compref.Ref{
//	    Repository: ociRepoSpec,
//	    Component:  "github.com/my-org/backend",
//	    Version:    "2.1.0",
//	}
//	descriptor, err := client.Get(ctx, ref)
//
//	// Get a component using a version alias
//	ref := compref.Ref{
//	    Repository: ociRepoSpec,
//	    Component:  "github.com/my-org/backend",
//	    Version:    "latest",  // or any other alias like "stable", "dev"
//	}
//	descriptor, err := client.Get(ctx, ref)
//
// The Get method returns ErrInvalidComponentReference if the reference is incomplete,
// and returns ErrNotFound if the component or version/alias doesn't exist.
//
// # Writing Component Descriptors
//
// Use the Save method to persist component descriptors to a repository:
//
//	import (
//	    descruntime "ocm.software/open-component-model/bindings/go/descriptor/runtime"
//	    "ocm.software/open-component-model/bindings/go/runtime"
//	)
//
//	// Construct your component descriptor
//	descriptor := descruntime.Descriptor{
//	    Component: descruntime.Component{
//	        Name:    "github.com/my-org/frontend",
//	        Version: "1.5.0",
//	        // ... other fields
//	    },
//	}
//
//	// Define the target repository
//	repoSpec := runtime.Typed{
//	    Type: "oci",
//	    Data: map[string]interface{}{
//	        "baseUrl": "ghcr.io/my-org/components",
//	    },
//	}
//
//	// Persist the descriptor
//	err := client.Save(ctx, repoSpec, descriptor)
//	if errors.Is(err, repository.ErrComponentAlreadyExists) {
//	    // Handle duplicate - idempotent operations can safely ignore this
//	}
//
// The Save method returns ErrComponentAlreadyExists if an identical component version
// already exists in the repository, ensuring immutability of released components.
//
// # Version Aliasing
//
// Version aliases provide human-readable names for component versions, allowing you to
// reference versions using names like "latest", "stable", or "production" instead of
// semantic version numbers.
//
// Use the AddAlias method to create or update version aliases:
//
//	// Create a reference to a specific component version
//	ref := compref.Ref{
//	    Repository: ociRepoSpec,
//	    Component:  "github.com/my-org/backend",
//	    Version:    "2.1.0",
//	}
//
//	// Add or update the "latest" alias to point to version 2.1.0
//	err := client.AddAlias(ctx, ref, "latest")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Users can now fetch the component using the alias
//	aliasRef := compref.Ref{
//	    Repository: ociRepoSpec,
//	    Component:  "github.com/my-org/backend",
//	    Version:    "latest",
//	}
//	descriptor, err := client.Get(ctx, aliasRef)
//
// Aliases are mutable - calling AddAlias with an existing alias name will update it
// to point to the new version. This is useful for maintaining rolling aliases like
// "latest" that should always point to the most recent release.
//
// # Error Handling
//
// The package defines three sentinel errors for common scenarios:
//
//   - ErrNotFound: returned when a component or version is not found
//   - ErrComponentAlreadyExists: returned when attempting to save a duplicate component version
//   - ErrInvalidComponentReference: returned when a component reference is invalid (e.g., missing version)
//
// # Advanced Configuration
//
// For use cases requiring direct control over credential resolution or repository providers,
// you can construct an OciClient directly using NewOciClient:
//
//	import (
//	    "ocm.software/open-component-model/bindings/go/credentials"
//	    "ocm.software/open-component-model/bindings/go/oci/repository/provider"
//	    "ocm.software/open-component-model/bindings/go/oci/repository/resource"
//	    "ocm.software/open-component-model/bindings/go/transfer"
//	)
//
//	resolver := credentials.NewResolver(/* custom config */)
//	repoProvider := provider.NewComponentVersionRepositoryProvider()
//	transferExecutor := repository.NewDefaultTransferExecutor(
//	    transfer.NewDefaultBuilder(repoProvider, resource.NewResourceRepository(nil), resolver),
//	)
//
//	client := repository.NewOciClient(
//	    resolver,
//	    repoProvider,
//	    transferExecutor,
//	    repository.WithOciClientLogger(logger),
//	)
//
// # See Also
//
//   - OCM Specification: https://github.com/open-component-model/ocm-spec
//   - OCM Website: https://ocm.software
//   - Open Component Model Project: https://github.com/open-component-model/open-component-model
package repository
