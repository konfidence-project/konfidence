package lrucache

import (
	"context"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"io"

	lru "github.com/hashicorp/golang-lru/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DefaultCacheSize is the headroom of entries held before the oldest is evicted.
const DefaultCacheSize = 2048

// Factory builds the value for one cache key on a miss. It is called with:
//   - ctx: the reconcile context — use logf.FromContext(ctx) for a logger
//     already enriched with cluster/namespace context.
//   - k8sClient: the cluster-scoped client for this reconcile — must not be
//     closed over at setup time as it is scoped to the logical cluster.
//   - in: the input value the cache key was derived from (e.g. the live CR, or
//     a plain string such as a vector version reference).
//
// Only truly process-wide state (e.g. Limiter) should be closed over.
// The result is stored by Lookup; the factory has no knowledge of caching.
type Factory[In, V any] func(ctx context.Context, k8sClient client.Reader, in In) (V, error)

// Cache is a process-wide generic LRU. Entries are evicted automatically when
// the LRU reaches capacity.
//
// In is the input type the key is derived from; V is the caller-defined value.
//
// Cache is safe for concurrent use.
type Cache[In, V any] struct {
	lru     *lru.Cache[uint64, V]
	extract func(in In) uint64
	factory Factory[In, V]
}

// New returns a Cache. extract derives the cache key from an input value;
// factory builds the value on a miss. Both are registered once at
// SetupControllers time.
func New[In, V any](size int, extract func(in In) uint64, factory Factory[In, V]) (*Cache[In, V], error) {
	l, err := lru.New[uint64, V](size)
	if err != nil {
		return nil, fmt.Errorf("lrucache: create LRU: %w", err)
	}
	return &Cache[In, V]{lru: l, extract: extract, factory: factory}, nil
}

// Lookup returns the cached V for this input, calling factory and storing the
// result on a miss. The cache key is derived via the extract func registered at
// New time.
//
// Hit  → cached V returned immediately; factory is not called.
// Miss → factory(ctx, k8sClient, in) called; result stored in LRU and returned.
//
// For CR inputs, a generation change produces a new key → natural miss. The old
// entry becomes unreachable and is evicted under LRU pressure. No explicit
// invalidation.
func (c *Cache[In, V]) Lookup(ctx context.Context, k8sClient client.Reader, in In) (V, error) {
	key := c.extract(in)
	if v, ok := c.lru.Get(key); ok {
		return v, nil
	}
	v, err := c.factory(ctx, k8sClient, in)
	if err != nil {
		var zero V
		return zero, err
	}
	c.lru.Add(key, v)
	return v, nil
}

// KeyableCR is satisfied by any controller-runtime CR — all CR types embed
// metav1.ObjectMeta which provides these three methods. Its generation makes
// it a natural, self-invalidating cache key: a spec change bumps generation,
// producing a fresh key and a natural miss.
type KeyableCR interface {
	GetNamespace() string
	GetName() string
	GetGeneration() int64
}

// CRExtract is a ready-made key extractor for CR types that embed
// metav1.ObjectMeta. Pass it directly to New:
//
//	lrucache.New(size, lrucache.CRExtract[*v1alpha1.VectorTemplate], factory)
func CRExtract[CR KeyableCR](cr CR) uint64 {
	return HashKey(cr.GetNamespace(), cr.GetName(), cr.GetGeneration())
}

// HashKey hashes the standard CR cache key fields via FNV-64a.
// Controllers that need no custom key logic use this via CRExtract.
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

// HashString hashes a single string key via FNV-64a. Use it to build an extract
// func for string-keyed caches (e.g. keyed by a vector version reference).
func HashString(s string) uint64 {
	h := fnv.New64a()
	_, _ = io.WriteString(h, s)
	return h.Sum64()
}
