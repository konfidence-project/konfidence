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
//	func verify(ctx context.Context, resolver credentials.Resolver, descriptors []*descruntime.Descriptor) error {
//	    // The verifier is stateless and safe to build once and share process-wide.
//	    // Specs and the resolver are passed per Verify call, not captured at build time.
//	    verifier, err := crypto.NewVerifierBuilder().
//	        WithParallelism(crypto.NewLimiter(0)). // optional: bound concurrent crypto
//	        WithCache(1024, 30*time.Minute).       // optional: memoize successful verdicts
//	        Build()
//	    if err != nil {
//	        return err
//	    }
//	    specs := []crypto.SignatureSpec{crypto.DefaultSignatureSpec("my-sig", nil)}
//	    return verifier.Verify(ctx, resolver, specs, descriptors)
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
// # Verification
//
// The Verifier verifies OCM descriptor signatures using:
//   - System certificate trust store (always loaded into the underlying RSA handler)
//   - An optional credentials.Resolver supplying additional anchors / public keys
//
// The resolver is optional. Without one, PEM signatures verify against the system
// trust store and plain signatures fail with the upstream ErrMissingPublicKey.
// When a resolver is configured, ErrNotFound is treated as "no creds for this
// signature": the handler still falls back to system roots for PEM. Any other
// resolver error is propagated and fails the whole verification.
//
// # Signing
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
//   - pkg/ocm/credentials.ResolverFromCredentials — translates *konfidence.Credentials
//     to []Ref and calls ResolverFromRefs.
//   - credentials.NewStaticCredentialsResolver — simple in-memory map, useful in tests.
//
// # Disabling Signing/Verification
//
// Disabling is expressed by supplying no SignatureSpecs — there are no separate
// noop types. A Verifier called with an empty []SignatureSpec is a no-op
// (Verify returns nil); a Signer built with no specs is a no-op (Sign returns
// nil). In both cases construction still yields a real, safe-to-call instance,
// so call sites never need a nil check or a special type.
//
//	signer, _ := crypto.NewSignerBuilder().Build()   // no specs → Sign is a no-op
//	verifier, _ := crypto.NewVerifierBuilder().Build() // Verify with empty specs is a no-op
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
