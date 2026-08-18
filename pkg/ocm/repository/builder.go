package repository

import (
	"context"

	"github.com/go-logr/logr"
	"ocm.software/open-component-model/bindings/go/credentials"
	"ocm.software/open-component-model/bindings/go/oci/repository/provider"
	"ocm.software/open-component-model/bindings/go/oci/repository/resource"
	"ocm.software/open-component-model/bindings/go/transfer"
)

// OciClientBuilder constructs OciClient instances.
// Call NewOciClientBuilder, configure with With* methods, then call Build.
//
// A nil or absent resolver means anonymous access — suitable for public registries.
// Pass the resolver from pkg/ocm/credentials.ResolverFromCredentials for authenticated access.
//
// Example:
//
//	resolver, err := credentials.ResolverFromCredentials(ctx, k8sClient, cr.Namespace, cr.Spec.Credentials)
//	if err != nil {
//	    return fmt.Errorf("building credentials: %w", err)
//	}
//	client, err := NewOciClientBuilder().
//	    WithResolver(resolver).
//	    WithLogger(log).
//	    Build(ctx)
type OciClientBuilder struct {
	resolver credentials.Resolver
	log      logr.Logger
}

// NewOciClientBuilder creates a new OciClientBuilder with default settings.
func NewOciClientBuilder() *OciClientBuilder {
	return &OciClientBuilder{}
}

// WithResolver sets the credentials.Resolver used to authenticate OCI registry requests.
// A nil resolver is valid and produces anonymous access.
func (b *OciClientBuilder) WithResolver(r credentials.Resolver) *OciClientBuilder {
	b.resolver = r
	return b
}

// WithLogger configures structured logging for the OciClient.
func (b *OciClientBuilder) WithLogger(log logr.Logger) *OciClientBuilder {
	b.log = log
	return b
}

// Build constructs an OciClient. A nil resolver falls back to NoopCredentialResolver
// (anonymous access). All other wiring is handled here.
func (b *OciClientBuilder) Build(_ context.Context) (Client, error) {
	resolver := b.resolver
	if resolver == nil {
		resolver = NoopCredentialResolver{}
	}
	repoProvider := provider.NewComponentVersionRepositoryProvider()
	transferExecutor := NewDefaultTransferExecutor(
		transfer.NewDefaultBuilder(repoProvider, resource.NewResourceRepository(nil), resolver),
	)
	var opts []OciClientOption
	if !b.log.IsZero() {
		opts = append(opts, WithOciClientLogger(b.log))
	}
	return NewOciClient(resolver, repoProvider, transferExecutor, opts...), nil
}
