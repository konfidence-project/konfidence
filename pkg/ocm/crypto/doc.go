// Package crypto provides RSA signing and verification for Open Component Model (OCM) descriptors.
//
// # Overview
//
// Both the signer and the verifier resolve credentials per signature through the
// upstream OCM credentials.Resolver.
//
// # Quick Start: Verification
//
//	import (
//	    "context"
//
//	    descruntime "ocm.software/open-component-model/bindings/go/descriptor/runtime"
//	    "github.com/konfidence-project/konfidence/pkg/ocm/crypto"
//	)
//
//	func verify(ctx context.Context, resolver credentials.Resolver, descriptors ...*descruntime.Descriptor) error {
//	    verifier, err := crypto.NewVerifierBuilder().
//	        WithSpecs([]crypto.SignatureSpec{crypto.DefaultSignatureSpec("my-sig", nil)}).
//	        WithResolver(resolver).
//	        Build()
//	    if err != nil {
//	        return err
//	    }
//	    return verifier.Verify(ctx, descriptors...)
//	}
//
// # Quick Start: Signing
//
//	func sign(ctx context.Context, resolver credentials.Resolver, descriptor *descruntime.Descriptor) error {
//	    signer, err := crypto.NewSignerBuilder().
//	        WithSpecs([]crypto.SignatureSpec{crypto.DefaultSignatureSpec("my-sig", nil)}).
//	        WithResolver(resolver).
//	        Build()
//	    if err != nil {
//	        return err
//	    }
//	    return signer.Sign(ctx, descriptor)
//	}
//
// # OCMVerifier
//
// OCMVerifier verifies OCM descriptor signatures using:
//   - System certificate trust store (always loaded into the underlying RSA handler)
//   - An optional credentials.Resolver supplying additional anchors / public keys
//
// The resolver is optional. Without one, PEM signatures verify against the system
// trust store and plain signatures fail with the upstream ErrMissingPublicKey.
// When a resolver is configured, ErrNotFound is treated as "no creds for this
// signature": the handler still falls back to system roots for PEM. Any other
// resolver error is propagated and fails the whole verification.
//
// # OCMSigner
//
// A non-nil credentials.Resolver is required. Any error from the resolver —
// including ErrNotFound — aborts the signing operation.
//
// # Credentials
//
// Both types take any implementation of
// ocm.software/open-component-model/bindings/go/credentials.Resolver. Common wirings:
//
//   - pkg/ocm/credentials.ResolverFromRefs — builds a graph from one or more
//     Secrets/ConfigMaps holding .ocmconfig data.
//   - pkg/ocm/credentials.ResolverFromCredentials — galaxy domain mapper; translates
//     *galaxy.Credentials to []Ref and calls ResolverFromRefs.
//   - credentials.NewStaticCredentialsResolver — simple in-memory map, useful in tests.
//
// # Disabling Signing/Verification
//
// Use the noop implementations to disable an operation entirely:
//
//	signer := crypto.NoopSigner{}    // Sign returns nil, no-op
//	verifier := crypto.NoopVerifier{} // Verify returns nil, no-op
//
// Both builders return the appropriate noop when WithSpecs receives an empty slice.
//
// # Error Handling
//
// Signing errors:
//   - Duplicate signature name already present on the descriptor
//   - Resolver error (including ErrNotFound — fail-fast)
//
// Verification errors:
//   - Required signature not found in descriptor
//   - Digest mismatch
//   - Resolver error other than ErrNotFound
//   - RSA handler errors (e.g. untrusted certificate, signature mismatch,
//     ErrMissingPublicKey for plain signatures without creds)
package crypto
