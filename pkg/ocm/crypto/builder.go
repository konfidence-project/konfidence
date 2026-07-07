package crypto

import (
	"fmt"

	"github.com/go-logr/logr"
	"ocm.software/open-component-model/bindings/go/credentials"
)

// VerifierBuilder constructs a Verifier from a set of SignatureSpecs and a credentials.Resolver.
// Call NewVerifierBuilder, configure with With* methods, then call Build.
type VerifierBuilder struct {
	specs    []SignatureSpec
	resolver credentials.Resolver
	limiter  Limiter
	log      logr.Logger
}

func NewVerifierBuilder() *VerifierBuilder {
	return &VerifierBuilder{
		log:     logr.Discard(),
		limiter: NewLimiter(0),
	}
}

func (b *VerifierBuilder) WithSpecs(specs []SignatureSpec) *VerifierBuilder {
	b.specs = specs
	return b
}

func (b *VerifierBuilder) WithResolver(r credentials.Resolver) *VerifierBuilder {
	b.resolver = r
	return b
}

func (b *VerifierBuilder) WithLimiter(l Limiter) *VerifierBuilder {
	b.limiter = l
	return b
}

func (b *VerifierBuilder) WithLogger(log logr.Logger) *VerifierBuilder {
	b.log = log
	return b
}

// Build returns a Verifier. Empty specs produce a NoopVerifier.
func (b *VerifierBuilder) Build() (Verifier, error) {
	if len(b.specs) == 0 {
		return NoopVerifier{}, nil
	}
	inner, err := NewOCMVerifier(b.resolver, b.specs,
		WithVerifierLogger(b.log),
		WithVerifierLimiter(b.limiter),
	)
	if err != nil {
		return nil, fmt.Errorf("VerifierBuilder.Build: %w", err)
	}
	return NewCachingVerifier(inner, DefaultVerifierCacheSize, DefaultVerifierCacheTTL,
		WithCachingVerifierLogger(b.log),
	), nil
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

// Build returns a Signer. Empty specs produce a NoopSigner.
// Non-empty specs with a nil resolver return an error — signing requires credentials.
func (b *SignerBuilder) Build() (Signer, error) {
	if len(b.specs) == 0 {
		return NoopSigner{}, nil
	}
	if b.resolver == nil {
		return nil, fmt.Errorf("SignerBuilder.Build: resolver is required when specs are provided")
	}
	signer, err := NewOCMSigner(b.resolver, b.specs,
		WithSignerLogger(b.log),
		WithSignerLimiter(b.limiter),
	)
	if err != nil {
		return nil, fmt.Errorf("SignerBuilder.Build: %w", err)
	}
	return signer, nil
}
