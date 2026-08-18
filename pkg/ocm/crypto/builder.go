package crypto

import (
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"ocm.software/open-component-model/bindings/go/credentials"
)

// VerifierBuilder configures and constructs a [Verifier].
//
// By default, the verifier performs sequential signature verification with no
// result caching. Use [VerifierBuilder.WithParallelism] and
// [VerifierBuilder.WithCache] to opt in to concurrent verification and
// caching of successful results respectively.
type VerifierBuilder struct {
	limiter   Limiter
	cacheSize int
	cacheTTL  time.Duration
	log       logr.Logger
}

// NewVerifierBuilder returns a builder with no parallelism and no caching enabled.
func NewVerifierBuilder() *VerifierBuilder {
	return &VerifierBuilder{
		log: logr.Discard(),
	}
}

// WithParallelism enables concurrent verification of the spec × descriptor
// matrix, bounded by the provided [Limiter]. Without this option each cell is
// verified sequentially.
func (b *VerifierBuilder) WithParallelism(l Limiter) *VerifierBuilder {
	b.limiter = l
	return b
}

// WithLogger attaches a logger that the verifier uses for diagnostic output.
func (b *VerifierBuilder) WithLogger(log logr.Logger) *VerifierBuilder {
	b.log = log
	return b
}

// WithCache enables caching of successful verification results. size is the
// maximum number of entries in the cache; ttl is how long a successful result
// is considered fresh before re-verification is required.
func (b *VerifierBuilder) WithCache(size int, ttl time.Duration) *VerifierBuilder {
	b.cacheSize = size
	b.cacheTTL = ttl
	return b
}

// Build constructs the [Verifier]. The returned instance is safe for
// concurrent use and should be shared process-wide.
func (b *VerifierBuilder) Build() (Verifier, error) {
	ocmVerifier, err := newOCMVerifier(
		withVerifierLogger(b.log),
	)
	if err != nil {
		return nil, fmt.Errorf("VerifierBuilder.Build: %w", err)
	}
	var inner Verifier = ocmVerifier
	if b.cacheSize > 0 {
		inner = newCachingVerifier(inner, b.cacheSize, b.cacheTTL, withCachingVerifierLogger(b.log))
	}
	if b.limiter != nil {
		inner = newParallelVerifier(inner, b.limiter)
	}
	return newPreFlightVerifier(inner), nil
}

// SignerBuilder constructs a Signer from a set of SignatureSpecs and a credentials.Resolver.
// Call NewSignerBuilder, configure with With* methods, then call Build.
type SignerBuilder struct {
	specs    []SignatureSpec
	resolver credentials.Resolver
	limiter  Limiter
	log      logr.Logger
}

func NewSignerBuilder() *SignerBuilder {
	return &SignerBuilder{
		log:     logr.Discard(),
		limiter: NewLimiter(0),
	}
}

func (b *SignerBuilder) WithSpecs(specs []SignatureSpec) *SignerBuilder {
	b.specs = specs
	return b
}

func (b *SignerBuilder) WithResolver(r credentials.Resolver) *SignerBuilder {
	b.resolver = r
	return b
}

func (b *SignerBuilder) WithLimiter(l Limiter) *SignerBuilder {
	b.limiter = l
	return b
}

func (b *SignerBuilder) WithLogger(log logr.Logger) *SignerBuilder {
	b.log = log
	return b
}

// Build returns a Signer. Empty specs produce a no-op signer (Sign returns nil),
// mirroring how a Verifier treats empty specs — this is the "signing disabled"
// state. Non-empty specs with a nil resolver return an error — signing requires
// credentials.
func (b *SignerBuilder) Build() (Signer, error) {
	resolver := b.resolver
	if len(b.specs) == 0 {
		// No specs → no-op signer. newOCMSigner requires a non-nil resolver, but
		// a disabled signer never resolves anything; supply an empty static
		// resolver so construction succeeds without demanding credentials the
		// caller has no reason to provide.
		if resolver == nil {
			resolver = credentials.NewStaticCredentialsResolver(nil)
		}
	} else if resolver == nil {
		return nil, fmt.Errorf("SignerBuilder.Build: resolver is required when specs are provided")
	}
	signer, err := newOCMSigner(resolver, b.specs,
		withSignerLogger(b.log),
		withSignerLimiter(b.limiter),
	)
	if err != nil {
		return nil, fmt.Errorf("SignerBuilder.Build: %w", err)
	}
	return signer, nil
}
