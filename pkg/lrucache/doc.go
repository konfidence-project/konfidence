// Package lrucache provides a process-wide, concurrency-safe generic LRU cache
// for controller-runtime reconcilers.
//
// # Why This Exists
//
// A controller reconciles the same inputs hundreds of times over their
// lifetime. Two distinct kinds of work benefit from caching across reconciles:
//
//   - Client bundles keyed by CR generation. A complete set of OCM clients
//     (credentials.Resolver, OCMVerifier with its CachingVerifier LRU,
//     OciClient, OCMSigner) is fully determined by the CR generation. Rebuilding
//     them every reconcile throws away the CachingVerifier's warm TTL window and
//     pays full construction cost each time.
//
//   - Immutable content keyed by an opaque string. A resolved, signature-verified
//     vector is fully determined by its concrete version reference (versions are
//     monotonically generated and never reused). Re-fetching it from OCI on every
//     reconcile is a pointless network round trip for the common no-drift case.
//
// [Cache] serves both: it is a generic LRU keyed by a uint64 hash. Two extract
// helpers cover the two shapes:
//
//   - [CRExtract] maps a CR → uint64 via namespace + name + generation. Any CR
//     that embeds metav1.ObjectMeta satisfies [KeyableCR] — no wrapper needed.
//     A spec change bumps generation, changing the key: a natural, self-invalidating
//     miss.
//
//   - [HashString] maps a single string → uint64, for keying by an opaque,
//     immutable identifier such as a vector version reference.
//
// # The Factory
//
// [New] takes a factory called on every cache miss with the reconcile context,
// the k8sClient, and the input the key was derived from. It owns the full
// construction path — credential resolution and client instantiation for client
// bundles, or the verified fetch for content. Only truly process-wide state
// (e.g. a shared [crypto.Limiter]) is closed over; k8sClient is passed through
// [Cache.Lookup]; use logf.FromContext(ctx) for a context-enriched logger.
//
// [Cache.Lookup] is the only call-site in the reconcile loop:
//
//   - Hit  → returns the cached value immediately; factory is never called.
//   - Miss → calls factory, stores the result, returns it.
//
// Superseded entries become unreachable and are evicted under LRU capacity
// pressure — no explicit invalidation, no background goroutines, no TTL.
//
// # Thread Safety
//
// [Cache] is safe for concurrent use.
//
// # Example — client bundle keyed by CR generation
//
//	cache, err := lrucache.New(
//	    lrucache.DefaultCacheSize,
//	    lrucache.CRExtract[*v1alpha1.VectorTemplate],
//	    func(ctx context.Context, k8sClient client.Reader, cr *v1alpha1.VectorTemplate) (vectorocm.Adapter, error) {
//	        resolver, err := credentials.ResolverFromCredentials(ctx, k8sClient, cr.Namespace, cr.Spec.Credentials)
//	        if err != nil {
//	            return vectorocm.Adapter{}, err
//	        }
//	        // ... build ociClient / verifiers / signer from resolver ...
//	        return vectorocm.NewAdapter(/* ... */), nil
//	    },
//	)
//
// Then in the reconcile loop — one call, everything warm:
//
//	adapter, err := r.cache.Lookup(ctx, k8sClient, template)
//
// # Example — verified content keyed by version reference
//
//	vectors, err := lrucache.New(
//	    lrucache.DefaultCacheSize,
//	    lrucache.HashString,
//	    func(ctx context.Context, _ client.Reader, ref string) (vector.Vector, error) {
//	        parsed, err := compref.Parse(ref)
//	        if err != nil {
//	            return vector.Vector{}, err
//	        }
//	        return adapter.GetVector(ctx, *parsed) // verifies signature on every miss
//	    },
//	)
package lrucache