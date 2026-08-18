package repository

import (
	"context"

	"ocm.software/open-component-model/bindings/go/credentials"
	"ocm.software/open-component-model/bindings/go/runtime"
)

var (
	_ credentials.Resolver = NoopCredentialResolver{}
)

// NoopCredentialResolver is a credentials.Resolver that always returns nil credentials
// without error.
//
// This resolver is used when no authentication is configured, allowing clients to
// attempt anonymous access to repositories.
//
// # Use Cases
//
//   - Public registries that don't require authentication
//   - Development and testing with local registries
//   - Default behavior when no secret is provided to OciClientBuilder
//
// # Behavior
//
// Resolve always returns (nil, nil) regardless of the identity parameter. This causes
// OciClient to attempt anonymous access to all repositories. If a repository requires
// authentication, the registry will return a 401/403 error.
type NoopCredentialResolver struct{}

// Resolve always returns nil credentials without error, indicating anonymous access.
//
// The identity parameter is ignored. All credential lookups return no credentials,
// which causes the OCI client to attempt anonymous access to repositories.
func (n NoopCredentialResolver) Resolve(_ context.Context, _ runtime.Identity) (runtime.Typed, error) {
	return nil, nil
}
