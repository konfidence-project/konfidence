package crypto

//go:generate go run go.uber.org/mock/mockgen -destination=internal/mocks/mock_ocm_verifier.go -package=mocks ocm.software/open-component-model/bindings/go/signing Verifier
//go:generate go run go.uber.org/mock/mockgen -destination=internal/mocks/mock_ocm_resolver.go -package=mocks ocm.software/open-component-model/bindings/go/credentials Resolver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/go-logr/logr"
	"golang.org/x/sync/errgroup"
	"ocm.software/open-component-model/bindings/go/credentials"
	ocm "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	rsahandler "ocm.software/open-component-model/bindings/go/rsa/signing/handler"
	"ocm.software/open-component-model/bindings/go/runtime"
	"ocm.software/open-component-model/bindings/go/signing"
)

var (
	_                             Verifier = (*OCMVerifier)(nil)
	_                             Verifier = (*NoopVerifier)(nil)
	verifyDigestMatchesDescriptor          = signing.VerifyDigestMatchesDescriptor
)

// Verifier is an interface for verifying OCM descriptors.
type Verifier interface {
	// Verify verifies the signatures of the given OCM descriptors.
	// If any verification fails, an error is returned.
	// Pending crypto work is aborted asap.
	Verify(ctx context.Context, descs ...*ocm.Descriptor) error
}

// OCMVerifier verifies OCM descriptors against a configurable set of SignatureSpecs.
// Credentials are resolved per signature through an optional credentials.Resolver. Without
// a resolver, PEM signatures fall back to the system trust store and plain signatures fail.
type OCMVerifier struct {
	log         logr.Logger
	resolver    credentials.Resolver
	rsaVerifier signing.Verifier
	specs       []SignatureSpec
	limiter     Limiter
}

// OCMVerifierOption configures an OCMVerifier.
type OCMVerifierOption func(*OCMVerifier)

// WithVerifierLogger sets the logger for the verifier.
// The constructor automatically appends signature names as log values.
func WithVerifierLogger(log logr.Logger) OCMVerifierOption {
	return func(v *OCMVerifier) {
		v.log = log
	}
}

// WithNamedVerifierLogger decorates the logger with the standard verifier name "ocm-verifier".
// The constructor automatically appends signature names as log values.
func WithNamedVerifierLogger(log logr.Logger) OCMVerifierOption {
	return func(v *OCMVerifier) {
		v.log = log.WithName("ocm-verifier")
	}
}

// WithVerifierLimiter installs a Limiter that bounds the number of concurrent
// verification operations. Pass the same Limiter to every Signer and Verifier
// in the process to share the budget. Without this option a NoopLimiter is used
// — verifications run unbounded.
func WithVerifierLimiter(l Limiter) OCMVerifierOption {
	return func(v *OCMVerifier) {
		if l != nil {
			v.limiter = l
		}
	}
}

func defaultOCMVerifierOptions() *OCMVerifier {
	return &OCMVerifier{
		log:     logr.Discard(),
		limiter: NoopLimiter{},
	}
}

// NewOCMVerifier creates a new OCMVerifier instance.
// At least one SignatureSpec must be provided.
func NewOCMVerifier(resolver credentials.Resolver, specs []SignatureSpec, opts ...OCMVerifierOption) (*OCMVerifier, error) {
	if err := specPreFlightSanityCheck(specs); err != nil {
		return nil, fmt.Errorf("create verifier: %w", err)
	}
	rsaHandler, err := rsahandler.New(runtime.NewScheme(), true) // load system roots
	if err != nil {
		return nil, fmt.Errorf("create rsa handler: %w", err)
	}
	o := defaultOCMVerifierOptions()
	for _, opt := range opts {
		opt(o)
	}
	names := make([]string, len(specs))
	for i, s := range specs {
		names[i] = s.Name
	}
	o.log = o.log.WithValues("signatures", fmt.Sprintf("%v", names))
	o.rsaVerifier = rsaHandler
	if resolver == nil {
		resolver = credentials.NewStaticCredentialsResolver(nil)
	}
	o.resolver = resolver
	o.specs = specs
	return o, nil
}

func (o *OCMVerifier) Verify(ctx context.Context, descs ...*ocm.Descriptor) error {
	if len(descs) == 0 {
		return nil
	}
	if len(descs) == 1 {
		return o.verify(ctx, descs[0])
	}
	// errgroup solves error aggregation and fail-fast: any descriptor failure
	// cancels gctx, so siblings short-circuit at the next ctx-aware operation
	// (Limiter.Acquire, credential resolution). Per-call SetLimit is intentionally
	// absent — concurrency is bounded process-wide by the Limiter installed via
	// WithVerifierLimiter.
	verifierPool, gctx := errgroup.WithContext(ctx)
	for _, t := range descs {
		verifierPool.Go(func() error { return o.verify(gctx, t) })
	}
	return verifierPool.Wait()
}

func (o *OCMVerifier) verify(ctx context.Context, desc *ocm.Descriptor) error {
	release, err := o.limiter.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("ocm descriptor verification failed: %w", err)
	}
	defer release()

	if err := isSafelyDigestible(&desc.Component); err != nil {
		return fmt.Errorf("ocm descriptor verification failed: descriptor is not safely digestible: %w", err)
	}
	for _, spec := range o.specs {
		if err := o.verifySignature(ctx, desc, spec); err != nil {
			return fmt.Errorf("ocm descriptor verification failed: %w", err)
		}
	}
	return nil
}

func (o *OCMVerifier) verifySignature(ctx context.Context, desc *ocm.Descriptor, spec SignatureSpec) error {
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
	creds, err := o.credsFor(ctx, sig)
	if err != nil {
		return fmt.Errorf("resolve credentials for %q: %w", spec.Name, err)
	}

	// Step 7: inject issuer if pinned (empty issuer rejected at construction by specPreFlightSanityCheck)
	if spec.Issuer != nil {
		sig.Signature.Issuer = *spec.Issuer
	}

	// Step 8: verify signature
	if err := o.rsaVerifier.Verify(ctx, sig, nil, creds); err != nil {
		return fmt.Errorf("signature verification failed for %q: %w", spec.Name, err)
	}

	return nil
}

// credsFor resolves credentials for the given signature through the configured resolver.
// ErrNotFound yields (nil, nil), letting the verifier fall back to its system trust roots for PEM.
func (o *OCMVerifier) credsFor(ctx context.Context, sig ocm.Signature) (runtime.Typed, error) {
	id, err := o.rsaVerifier.GetVerifyingCredentialConsumerIdentity(ctx, sig, nil)
	if err != nil {
		return nil, fmt.Errorf("derive consumer identity: %w", err)
	}
	creds, err := o.resolver.Resolve(ctx, id)
	if errors.Is(err, credentials.ErrNotFound) {
		return nil, nil
	}
	return creds, err
}

// specPreFlightSanityCheck validates SignatureSpecs: non-empty slice, non-empty names, no duplicate names,
// and no spec with a non-nil empty issuer pin.
func specPreFlightSanityCheck(specs []SignatureSpec) error {
	if len(specs) == 0 {
		return fmt.Errorf("at least one signature spec must be provided")
	}
	seen := make(map[string]struct{}, len(specs))
	for _, spec := range specs {
		if strings.TrimSpace(spec.Name) == "" {
			return fmt.Errorf("signature names cannot be empty or whitespace")
		}
		if _, exists := seen[spec.Name]; exists {
			return fmt.Errorf("duplicate signature name detected: %q", spec.Name)
		}
		seen[spec.Name] = struct{}{}
		if spec.Issuer != nil && *spec.Issuer == "" {
			return fmt.Errorf("issuer pin for %q must not be empty; use nil to disable issuer pinning", spec.Name)
		}
	}
	return nil
}

// NoopVerifier is a Verifier implementation that does not perform any verification and returns nil for all operations.
// It's the goto way to disable verification.
type NoopVerifier struct{}

func (n NoopVerifier) Verify(ctx context.Context, descs ...*ocm.Descriptor) error {
	return nil
}
