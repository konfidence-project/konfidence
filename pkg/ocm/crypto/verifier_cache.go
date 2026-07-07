package crypto

import (
	"context"
	"fmt"
	"hash/fnv"
	"io"
	"log/slog"
	"time"

	"github.com/go-logr/logr"
	lru "github.com/hashicorp/golang-lru/v2/expirable"
	"golang.org/x/sync/errgroup"
	ocm "ocm.software/open-component-model/bindings/go/descriptor/runtime"
)

const (
	// DefaultVerifierCacheTTL is the maximum time a successful verification result
	// is served from cache without re-running the crypto. Bounded staleness: a cert
	// that expires mid-TTL gets a free pass for at most this duration — consistent
	// with the upstream OCM controller's 30-minute resolver cache.
	DefaultVerifierCacheTTL = 30 * time.Minute

	// DefaultVerifierCacheSize is the maximum number of descriptors held in the
	// LRU before oldest entries are evicted.
	DefaultVerifierCacheSize = 1024
)

// CachingVerifier is a Verifier decorator that memorizes successful verifications.
//
// All descriptors in a batch are verified concurrently. Each descriptor is cached
// independently: a sibling failure does not cancel in-flight work — every descriptor
// that passes is written to the cache regardless of what others do. The caller's
// context is passed directly to the inner Verifier so that an external cancellation
// (reconcile timeout, controller shutdown) still propagates, but a sibling failure
// does not short-circuit remaining work.
//
// Failures are never cached. A transient credentials error or a verification that
// races a cert rotation will be retried on the next reconcile.
//
// The cache can be flushed at any time via Flush — use this when a signing key is
// known to have been compromised and stale verdicts must not be served for the
// remainder of the TTL window.
type CachingVerifier struct {
	inner Verifier
	log   logr.Logger
	cache *lru.LRU[uint64, struct{}]
}

// CachingVerifierOption configures a CachingVerifier.
type CachingVerifierOption func(*CachingVerifier)

// WithCachingVerifierLogger sets a logger.
func WithCachingVerifierLogger(log logr.Logger) CachingVerifierOption {
	return func(c *CachingVerifier) {
		c.log = log
	}
}

// NewCachingVerifier wraps inner with a per-descriptor LRU verification cache.
// size is the maximum number of cached descriptors; ttl is how long a successful
// verdict is considered fresh. Pass DefaultVerifierCacheSize / DefaultVerifierCacheTTL
// for the recommended defaults.
func NewCachingVerifier(inner Verifier, size int, ttl time.Duration, opts ...CachingVerifierOption) *CachingVerifier {
	c := &CachingVerifier{
		inner: inner,
		log:   logr.Discard(),
		cache: lru.NewLRU[uint64, struct{}](size, nil, ttl),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Verify checks each descriptor against the cache; descriptors with a fresh cached
// verdict are skipped. Uncached descriptors are forwarded to the inner Verifier
// concurrently. The caller's ctx is passed to each inner call — not a derived
// context — so a single failure does not cancel siblings. Every descriptor that
// passes is cached; the first error encountered is returned after all goroutines
// complete.
func (c *CachingVerifier) Verify(ctx context.Context, descs ...*ocm.Descriptor) error {
	if len(descs) == 0 {
		return nil
	}
	if len(descs) == 1 {
		return c.verifyOne(ctx, descs[0])
	}
	// Zero-value errgroup: no derived context. Sibling failures do not cancel each
	// other — ctx cancellation still propagates because each verifyOne call receives
	// the original caller ctx directly. This is intentional: we cache best-effort,
	// every descriptor that passes gets cached regardless of what siblings do.
	var g errgroup.Group
	for _, desc := range descs {
		g.Go(func() error { return c.verifyOne(ctx, desc) })
	}
	return g.Wait()
}

func (c *CachingVerifier) verifyOne(ctx context.Context, desc *ocm.Descriptor) error {
	// Pre-check: verify every signature's digest against the actual descriptor
	// content before consulting the cache. This binds the FNV cache key (which
	// includes sig.Digest.Value) to the real descriptor content via a
	// SHA-256/SHA-512 integrity check — preventing a replay attack where an
	// attacker copies a valid signature struct onto a different descriptor to
	// obtain a cache hit. VerifyDigestMatchesDescriptor rejects any hash
	// algorithm outside SHA-256/SHA-512, so weak-algorithm bypass is not possible.
	log := slog.New(logr.ToSlogHandler(c.log))
	for _, sig := range desc.Signatures {
		if err := verifyDigestMatchesDescriptor(ctx, desc, sig, log); err != nil {
			return fmt.Errorf("cache pre-check digest mismatch for %q: %w", sig.Name, err)
		}
	}

	key := cacheKey(desc)
	if _, hit := c.cache.Get(key); hit {
		return nil
	}
	if err := c.inner.Verify(ctx, desc); err != nil {
		return err
	}
	c.cache.Add(key, struct{}{})
	return nil
}

// Flush removes all entries from the cache. Use when a signing key is known to
// have been compromised and stale verdicts must not be served for the remainder
// of the TTL window.
func (c *CachingVerifier) Flush() {
	c.cache.Purge()
}

// cacheKey produces a FNV-64a hash over all signature wire values in the
// descriptor. The cache is in-process only; the security binding to descriptor
// content is provided by the SHA-256/SHA-512 digest pre-check in verifyOne —
// not by this hash function. FNV-64a is used purely for fast, low-collision
// bucketing across the LRU.
//
// Per signature (null-byte separated):
//   - sig.Name
//   - sig.Digest.{HashAlgorithm, NormalisationAlgorithm, Value}
//   - sig.Signature.{Algorithm, MediaType, Issuer, Value}
func cacheKey(desc *ocm.Descriptor) uint64 {
	h := fnv.New64a()
	sep := []byte{0}
	write := func(s string) {
		_, _ = io.WriteString(h, s)
		_, _ = h.Write(sep)
	}
	for _, sig := range desc.Signatures {
		write(sig.Name)
		write(sig.Digest.HashAlgorithm)
		write(sig.Digest.NormalisationAlgorithm)
		write(sig.Digest.Value)
		write(sig.Signature.Algorithm)
		write(sig.Signature.MediaType)
		write(sig.Signature.Issuer)
		write(sig.Signature.Value)
	}
	return h.Sum64()
}
