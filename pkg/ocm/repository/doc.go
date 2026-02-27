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
// # Authentication with Kubernetes Secrets
//
// For private OCI registries, provide a Kubernetes Secret containing Docker config JSON.
// The secret must be of type kubernetes.io/dockerconfigjson:
//
//	// Load your secret from the Kubernetes API or construct it
//	secret := &corev1.Secret{
//	    Type: corev1.SecretTypeDockerConfigJson,
//	    Data: map[string][]byte{
//	        corev1.DockerConfigJsonKey: []byte(`{"auths":{"registry.example.com":{"auth":"..."}}}`),
//	    },
//	}
//
//	client, err := repository.NewOciClientBuilder().
//	    WithDockerConfigJsonSecret(secret).
//	    WithLogger(logger).
//	    Build(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// The builder automatically:
//   - Parses the Docker config JSON from the secret
//   - Creates an OCM credential graph for authenticating with registries
//   - Configures the credential resolver to match registry hostnames with credentials
//
// If no credentials are found for a given registry, the client attempts anonymous access,
// which works for public registries.
//
// # Reading Component Descriptors
//
// Use the Get method to retrieve component descriptors by reference:
//
//	// Get a specific version
//	ref := compref.Ref{
//	    Repository: ociRepoSpec,
//	    Component:  "github.com/my-org/backend",
//	    Version:    "2.1.0",
//	}
//	descriptor, err := client.Get(ctx, ref)
//
//	// Get the latest version (omit Version field)
//	ref := compref.Ref{
//	    Repository: ociRepoSpec,
//	    Component:  "github.com/my-org/backend",
//	}
//	latest, err := client.Get(ctx, ref)
//
// The Get method returns ErrNotFound if the component or version doesn't exist.
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
// # Error Handling
//
// The package defines two sentinel errors for common scenarios:
//
//   - ErrNotFound: returned when a component or version is not found
//   - ErrComponentAlreadyExists: returned when attempting to save a duplicate component version
//
// # Advanced Configuration
//
// For use cases requiring direct control over credential resolution or repository providers,
// you can construct an OciClient directly using NewOciClient:
//
//	import (
//	    "ocm.software/open-component-model/bindings/go/credentials"
//	    "ocm.software/open-component-model/bindings/go/oci/repository/provider"
//	)
//
//	resolver := credentials.NewResolver(/* custom config */)
//	provider := provider.NewComponentVersionRepositoryProvider()
//
//	client := repository.NewOciClient(
//	    resolver,
//	    provider,
//	    repository.WithOciClientLogger(logger),
//	)
//
// # Thread Safety
//
// The OciClient is safe for concurrent use by multiple goroutines. All methods are
// safe to call concurrently, and the underlying credential resolver and repository
// provider are designed to handle concurrent access.
//
// # See Also
//
//   - OCM Specification: https://github.com/open-component-model/ocm-spec
//   - OCM Website: https://ocm.software
//   - Open Component Model Project: https://github.com/open-component-model/open-component-model
package repository
