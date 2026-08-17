package crypto

//go:generate go run go.uber.org/mock/mockgen -destination=internal/mocks/mock_ocm_verifier.go -package=mocks ocm.software/open-component-model/bindings/go/signing Verifier
//go:generate go run go.uber.org/mock/mockgen -destination=internal/mocks/mock_ocm_resolver.go -package=mocks ocm.software/open-component-model/bindings/go/credentials Resolver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/go-logr/logr"
	"ocm.software/open-component-model/bindings/go/credentials"
	ocm "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	rsahandler "ocm.software/open-component-model/bindings/go/rsa/signing/handler"
	"ocm.software/open-component-model/bindings/go/runtime"
	"ocm.software/open-component-model/bindings/go/signing"
)

var (
	_                             Verifier = (*ocmVerifier)(nil)
	verifyDigestMatchesDescriptor          = signing.VerifyDigestMatchesDescriptor
)

// Verifier verifies a matrix of OCM descriptors against a set of SignatureSpecs.
//
// The verifier holds no per-CR state; specs and credentials are supplied per
// call. This makes the same Verifier instance safe to share process-wide
// across every reconciler.
//
// Implementations must be safe for concurrent use.
type Verifier interface {
	// Verify checks that every descriptor in descs carries, for every spec in
	// specs, a signature named spec.Name that satisfies the spec (media type,
	// hash algorithm, normalisation algorithm, optional issuer pin) and passes
	// cryptographic verification via credentials resolved from resolver.
	// An empty specs or descs slice is a no-op.
	Verify(ctx context.Context, resolver credentials.Resolver, specs []SignatureSpec, descs []*ocm.Descriptor) error
}

type ocmVerifier struct {
	log         logr.Logger
	rsaVerifier signing.Verifier
}

type ocmVerifierOption func(*ocmVerifier)

func withVerifierLogger(log logr.Logger) ocmVerifierOption {
	return func(v *ocmVerifier) {
		v.log = log
	}
}

func defaultOCMVerifierOptions() *ocmVerifier {
	return &ocmVerifier{
		log: logr.Discard(),
	}
}

func newOCMVerifier(opts ...ocmVerifierOption) (*ocmVerifier, error) {
	rsaHandler, err := rsahandler.New(runtime.NewScheme(), true)
	if err != nil {
		return nil, fmt.Errorf("create rsa handler: %w", err)
	}
	o := defaultOCMVerifierOptions()
	for _, opt := range opts {
		opt(o)
	}
	o.rsaVerifier = rsaHandler
	return o, nil
}

func (o *ocmVerifier) Verify(ctx context.Context, resolver credentials.Resolver, specs []SignatureSpec, descs []*ocm.Descriptor) error {
	if len(specs) == 0 || len(descs) == 0 {
		return nil
	}
	if resolver == nil {
		resolver = credentials.NewStaticCredentialsResolver(nil)
	}
	for _, desc := range descs {
		if err := isSafelyDigestible(&desc.Component); err != nil {
			return fmt.Errorf("ocm descriptor verification failed: descriptor is not safely digestible: %w", err)
		}
		for _, spec := range specs {
			if err := o.verifySignature(ctx, resolver, desc, spec); err != nil {
				return err
			}
		}
	}
	return nil
}

func (o *ocmVerifier) verifySignature(ctx context.Context, resolver credentials.Resolver, desc *ocm.Descriptor, spec SignatureSpec) error {
	sigLog := slog.New(logr.ToSlogHandler(o.log))

	// Step 1: locate sig by spec.Name
	idx := slices.IndexFunc(desc.Signatures, func(sig ocm.Signature) bool { return sig.Name == spec.Name })
	if idx == -1 {
		return fmt.Errorf("signature %q not found in descriptor", spec.Name)
	}
	sig := desc.Signatures[idx]

	// Step 2: check media type
	if sig.Signature.MediaType != spec.MediaType {
		return fmt.Errorf("media type mismatch for %q: got %q, want %q", spec.Name, sig.Signature.MediaType, spec.MediaType)
	}

	// Step 3: check hash algorithm
	if sig.Digest.HashAlgorithm != spec.HashAlgorithm {
		return fmt.Errorf("hash algorithm mismatch for %q: got %q, want %q", spec.Name, sig.Digest.HashAlgorithm, spec.HashAlgorithm)
	}

	// Step 4: check normalisation algorithm
	if sig.Digest.NormalisationAlgorithm != spec.NormalisationAlgorithm {
		return fmt.Errorf("normalisation algorithm mismatch for %q: got %q, want %q", spec.Name, sig.Digest.NormalisationAlgorithm, spec.NormalisationAlgorithm)
	}

	// Step 5: verify digest matches descriptor
	if err := verifyDigestMatchesDescriptor(ctx, desc, sig, sigLog); err != nil {
		return fmt.Errorf("digest mismatch for %q: %w", spec.Name, err)
	}

	// Step 6: resolve credentials
	creds, err := o.credsFor(ctx, resolver, sig)
	if err != nil {
		return fmt.Errorf("resolve credentials for %q: %w", spec.Name, err)
	}

	// Step 7: inject issuer if pinned
	if spec.Issuer != nil {
		sig.Signature.Issuer = *spec.Issuer
	}

	// Step 8: verify signature
	if err := o.rsaVerifier.Verify(ctx, sig, nil, creds); err != nil {
		return fmt.Errorf("signature verification failed for %q: %w", spec.Name, err)
	}

	return nil
}

// credsFor resolves credentials for the given signature through the supplied resolver.
// ErrNotFound yields (nil, nil), letting the verifier fall back to its system trust roots for PEM.
func (o *ocmVerifier) credsFor(ctx context.Context, resolver credentials.Resolver, sig ocm.Signature) (runtime.Typed, error) {
	id, err := o.rsaVerifier.GetVerifyingCredentialConsumerIdentity(ctx, sig, nil)
	if err != nil {
		return nil, fmt.Errorf("derive consumer identity: %w", err)
	}
	creds, err := resolver.Resolve(ctx, id)
	if errors.Is(err, credentials.ErrNotFound) {
		return nil, nil
	}
	return creds, err
}
