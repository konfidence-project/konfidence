package clientcache

import (
	"context"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"io"

	lru "github.com/hashicorp/golang-lru/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DefaultClientCacheSize is the headroom of cluster-generation entries held before
// the oldest is evicted.
const DefaultClientCacheSize = 2048

// Factory builds the full set of OCM clients for one CR generation.
// It is called on every cache miss with:
//   - ctx: the reconcile context — use logf.FromContext(ctx) for a logger
//     already enriched with cluster/namespace context.
//   - k8sClient: the cluster-scoped client for this reconcile — must not be
//     closed over at setup time as it is scoped to the logical cluster.
//   - cr: the live CR object carrying credentials and spec fields.
//
// Only truly process-wide state (e.g. Limiter) should be closed over.
// The result is stored by Lookup; the factory has no knowledge of caching.
type Factory[CR, C any] func(ctx context.Context, k8sClient client.Reader, cr CR) (C, error)

// Cache is a process-wide LRU registry of OCM clients.
// Each entry holds the complete set of clients needed by one controller for
// one CR generation. Entries are evicted automatically when the LRU reaches capacity.
//
// CR is the CR type being reconciled. C is the caller-defined client bundle.
//
// Cache is safe for concurrent use.
type Cache[CR, C any] struct {
	lru     *lru.Cache[uint64, C]
	extract func(cr CR) uint64
	factory Factory[CR, C]
}

// New returns a Cache. extract derives the cache key from a CR; factory builds
// the client bundle on a miss. Both are registered once at SetupControllers time.
func New[CR, C any](size int, extract func(cr CR) uint64, factory Factory[CR, C]) (*Cache[CR, C], error) {
	l, err := lru.New[uint64, C](size)
	if err != nil {
		return nil, fmt.Errorf("clientcache: create LRU: %w", err)
	}
	return &Cache[CR, C]{lru: l, extract: extract, factory: factory}, nil
}

// Lookup returns the cached C for this CR, calling factory and storing the result
// on a miss. The cache key is derived via the extract func registered at New time.
//
// Hit  → cached C returned immediately; factory is not called.
// Miss → factory(ctx, k8sClient, cr) called; result stored in LRU and returned.
//
// A generation change in the CR produces a new key → natural miss. The old entry
// becomes unreachable at some point and is evicted under LRU pressure. No explicit invalidation.
func (c *Cache[CR, C]) Lookup(ctx context.Context, k8sClient client.Reader, cr CR) (C, error) {
	key := c.extract(cr)
	if v, ok := c.lru.Get(key); ok {
		return v, nil
	}
	v, err := c.factory(ctx, k8sClient, cr)
	if err != nil {
		var zero C
		return zero, err
	}
	c.lru.Add(key, v)
	return v, nil
}

// KeyableObject is satisfied by any controller-runtime object — all CR types
// embed metav1.ObjectMeta which provides these three methods.
type KeyableObject interface {
	GetNamespace() string
	GetName() string
	GetGeneration() int64
}

// DefaultExtract is a ready-made key extractor for CR types that embed
// metav1.ObjectMeta. Pass it directly to New:
//
//	clientcache.New(size, clientcache.DefaultExtract[*v1alpha1.VectorTemplate], factory)
func DefaultExtract[CR KeyableObject](cr CR) uint64 {
	return HashKey(cr.GetNamespace(), cr.GetName(), cr.GetGeneration())
}

// HashKey hashes the standard cache key fields via FNV-64a.
// Controllers that need no custom key logic use this via DefaultExtract.
func HashKey(namespace, name string, generation int64) uint64 {
	h := fnv.New64a()
	sep := []byte{0}
	write := func(s string) {
		_, _ = io.WriteString(h, s)
		_, _ = h.Write(sep)
	}
	write(namespace)
	write(name)
	_ = binary.Write(h, binary.LittleEndian, generation)
	return h.Sum64()
}
