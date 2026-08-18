package crypto

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	lru "github.com/hashicorp/golang-lru/v2/expirable"
	"ocm.software/open-component-model/bindings/go/credentials"
	ocm "ocm.software/open-component-model/bindings/go/descriptor/runtime"
)

// cachingVerifier is a Verifier decorator that memorizes successful
// per-(descriptor, signature, spec) verifications.
//
// The cache is process-wide when the same cachingVerifier is passed to every
// reconciler — different CRs verifying the same signature bytes under the same
// spec share a single verify + a single cache entry.
//
// Failures are never cached. A transient credentials error or a verification
// that races a cert rotation will be retried on the next reconcile.
//
// The cache can be flushed at any time via Flush — use this when a signing key
// is known to have been compromised and stale verdicts must not be served for
// the remainder of the TTL window.
type cachingVerifier struct {
	inner Verifier
	log   logr.Logger
	cache *lru.LRU[string, struct{}]
}

type cachingVerifierOption func(*cachingVerifier)

func withCachingVerifierLogger(log logr.Logger) cachingVerifierOption {
	return func(c *cachingVerifier) {
		c.log = log
	}
}

func newCachingVerifier(inner Verifier, size int, ttl time.Duration, opts ...cachingVerifierOption) *cachingVerifier {
	c := &cachingVerifier{
		inner: inner,
		log:   logr.Discard(),
		cache: lru.NewLRU[string, struct{}](size, nil, ttl),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *cachingVerifier) Verify(ctx context.Context, resolver credentials.Resolver, specs []SignatureSpec, descs []*ocm.Descriptor) error {
	if len(specs) == 0 || len(descs) == 0 {
		return nil
	}
	// Best-effort: verify every cell so the cache is filled for every success,
	// even when a sibling fails. Return the FIRST error after the full pass —
	// never short-circuit, so a single bad cell can't deny caching the good
	// ones. (When wrapped by parallelVerifier this loop sees one cell at a time,
	// but the same guarantee must hold if the cache is used without it.)
	var firstErr error
	for _, desc := range descs {
		for _, spec := range specs {
			if err := c.verify(ctx, resolver, spec, desc); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func (c *cachingVerifier) verify(ctx context.Context, resolver credentials.Resolver, spec SignatureSpec, desc *ocm.Descriptor) error {
	// Locate the signature named by spec.Name. Missing signatures fall through
	// to the inner verifier so its error message ("signature %q not found in
	// descriptor") is the single source of truth.
	idx := slices.IndexFunc(desc.Signatures, func(sig ocm.Signature) bool { return sig.Name == spec.Name })
	if idx == -1 {
		return c.inner.Verify(ctx, resolver, []SignatureSpec{spec}, []*ocm.Descriptor{desc})
	}
	sig := desc.Signatures[idx]

	// Pre-check: verify the target signature's digest against the actual
	// descriptor content before consulting the cache. This binds the cache
	// key (which includes sig.Digest.Value and sig.Signature.Value) to the
	// real descriptor content via a SHA-256/SHA-512 integrity check —
	// preventing a replay attack where an attacker copies a valid signature
	// struct onto a different descriptor to obtain a cache hit.
	// VerifyDigestMatchesDescriptor rejects any hash algorithm outside
	// SHA-256/SHA-512, so weak-algorithm bypass is not possible.
	log := slog.New(logr.ToSlogHandler(c.log))
	if err := verifyDigestMatchesDescriptor(ctx, desc, sig, log); err != nil {
		return fmt.Errorf("cache pre-check digest mismatch for %q: %w", sig.Name, err)
	}

	key := cacheKey(sig, spec)
	if _, hit := c.cache.Get(key); hit {
		return nil
	}
	if err := c.inner.Verify(ctx, resolver, []SignatureSpec{spec}, []*ocm.Descriptor{desc}); err != nil {
		return err
	}
	c.cache.Add(key, struct{}{})
	return nil
}

func (c *cachingVerifier) Flush() {
	c.cache.Purge()
}

// cacheKey serializes signature wire bytes plus the spec fingerprint into a
// length-prefixed string. Two calls collide on the map iff their key strings
// are byte-equal — a structural, not statistical, property. No hash of our
// own; Go's internal map hash operates on well-defined string equality.
//
// Each field is written as its byte length in decimal, a ':' terminator, then
// the raw field bytes: "<len>:<bytes>". This framing is unconditionally
// injective — the length tells the reader exactly how many bytes follow, so
// the field content may contain ANY byte (including ':' or '\x00') without
// ambiguity. A plain delimiter would instead have to assume its separator
// never appears in a field; length-prefixing removes that assumption. This is
// the standard domain-separation framing used by TLS/Noise/SSH for exactly
// this reason, and it matters here because non-collision is a security
// property: a lax-spec verdict must never be served to an issuer-pinned
// caller sharing the same signature bytes.
//
// The "pin"/"nopin" tag on the issuer field disambiguates "no pin configured"
// (spec.Issuer == nil, "nopin" written) from a configured pin (the pin value
// is length-prefixed and thus cannot alias the "nopin" sentinel).
//
// Two invariants keep this design safe; both must hold on every touch:
//  1. verifyDigestMatchesDescriptor (SHA-256/SHA-512) runs before cache lookup
//     in verify, binding signature bytes to descriptor content. An attacker
//     cannot craft a colliding cache entry without also producing a
//     legitimately-signed descriptor.
//  2. The cache value is struct{}{} — a boolean verdict, no attacker-useful
//     payload. A hypothetical key equality between two legitimately-verified
//     entries leaks nothing beyond "already verified."
//
// If either invariant is ever violated, this key design must be reconsidered.
//
// Per signature (each field length-prefixed):
//   - sig.Name
//   - sig.Digest.{HashAlgorithm, NormalisationAlgorithm, Value}
//   - sig.Signature.{Algorithm, MediaType, Issuer, Value}
//
// Per spec (each field length-prefixed):
//   - spec.Name
//   - spec.MediaType
//   - spec.HashAlgorithm
//   - spec.NormalisationAlgorithm
//   - "pin" + *spec.Issuer, or "nopin"
func cacheKey(sig ocm.Signature, spec SignatureSpec) string {
	var b strings.Builder
	b.Grow(512)
	// write frames each field as "<len>:<bytes>". The decimal length prefix,
	// terminated by ':', makes the encoding injective regardless of field
	// content — no byte is forbidden inside a field.
	write := func(s string) {
		b.WriteString(strconv.Itoa(len(s)))
		b.WriteByte(':')
		b.WriteString(s)
	}

	// signature wire bytes (bound to descriptor content via SHA-256/512 pre-check)
	write(sig.Name)
	write(sig.Digest.HashAlgorithm)
	write(sig.Digest.NormalisationAlgorithm)
	write(sig.Digest.Value)
	write(sig.Signature.Algorithm)
	write(sig.Signature.MediaType)
	write(sig.Signature.Issuer)
	write(sig.Signature.Value)

	// spec fingerprint — what the OCMVerifier enforces beyond crypto
	write(spec.Name)
	write(spec.MediaType)
	write(spec.HashAlgorithm)
	write(spec.NormalisationAlgorithm)
	if spec.Issuer != nil {
		write("pin")
		write(*spec.Issuer)
	} else {
		write("nopin")
	}
	return b.String()
}
