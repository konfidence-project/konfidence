// Package clientcache provides a process-wide LRU registry of OCM clients
// for controller-runtime reconcilers.
//
// # Why This Exists
//
// A controller reconciles the same CR hundreds of times over its lifetime.
// Without a durable client registry, each reconcile allocates a fresh
// credentials.Resolver, OCMVerifier (with its CachingVerifier LRU), OciClient,
// and OCMSigner — only to throw them away at the end of the iteration. The
// CachingVerifier's 30-minute TTL window is never realised; every reconcile
// starts cold and pays the full construction cost again.
//
// # The Core Invariant
//
// A complete set of OCM clients is fully determined by the CR generation (a
// monotonically increasing integer that increments on every spec change).
// If the CR hasn't changed since the last reconcile, the clients are identical
// to those built last time and can be reused without rebuilding.
//
// [Cache] encodes this invariant directly. It is a generic LRU keyed by a
// uint64 derived from namespace + name + generation. Controllers register two
// functions once at SetupControllers time:
//
//   - extract: maps (cr) → uint64 cache key. [DefaultExtract]
//     covers any CR that embeds metav1.ObjectMeta — no wrapper needed.
//
//   - factory: called on every cache miss with the reconcile context,
//     the k8sClient, and the live CR. It owns the full construction path:
//     credential resolution, spec extraction, and client instantiation.
//     Only truly process-wide state (e.g. a shared [crypto.Limiter]) is closed
//     over — k8sClient is passed through [Cache.Lookup]; use
//     logf.FromContext(ctx) for a logger already enriched with context values.
//
// [Cache.Lookup] is the only call-site in the reconcile loop:
//
//   - Hit  → returns the cached bundle immediately; factory is never called.
//   - Miss → calls factory, stores the result, returns it.
//
// A spec change increments generation, which changes the cache key and
// produces a natural miss. The superseded entry becomes unreachable and is
// evicted under LRU capacity pressure — no explicit invalidation required,
// no background goroutines, no TTL.
//
// # Thread Safety
//
// [Cache] is safe for concurrent use.
//
// # What the factory produces
//
// The factory's return type C is caller-defined. The canonical pattern is to
// produce the adapter type that the controller already depends on, with all
// fields populated from the CR's spec and resolved credentials. This avoids
// introducing an intermediate bundle type whose only purpose would be to carry
// values between the factory and the reconcile loop.
//
// # Example
//
// Register once at SetupControllers time, producing a fully-wired adapter:
//
//	cache, err := clientcache.New(
//	    clientcache.DefaultClientCacheSize,
//	    clientcache.DefaultExtract[*v1alpha1.VectorTemplate],
//	    func(ctx context.Context, k8sClient client.Reader, cr *v1alpha1.VectorTemplate) (vectorocm.Adapter, error) {
//	        resolver, err := credentials.ResolverFromCredentials(ctx, k8sClient, cr.Namespace, cr.Spec.Credentials)
//	        if err != nil {
//	            return vectorocm.Adapter{}, err
//	        }
//	        ociClient, err := repository.NewOciClientBuilder().
//	            WithResolver(resolver).WithLogger(log).Build(ctx)
//	        if err != nil {
//	            return vectorocm.Adapter{}, err
//	        }
//	        vectorVerifier, err := crypto.NewVerifierBuilder().
//	            WithSpecs(crypto.SpecsFromVerify(cr.Spec.VerifyVector)).
//	            WithResolver(resolver).WithLimiter(limiter).Build()
//	        if err != nil {
//	            return vectorocm.Adapter{}, err
//	        }
//	        // ... additional verifiers / signer built the same way ...
//	        return vectorocm.NewAdapter(
//	            vectorocm.WithOCMClient(ociClient),
//	            vectorocm.WithVectorVerifier(vectorVerifier),
//	        ), nil
//	    },
//	)
//
// Then in the reconcile loop — one call, everything warm:
//
//	adapter, err := r.cache.Lookup(ctx, k8sClient, template)
//	if err != nil {
//	    return fmt.Errorf("building OCM clients: %w", err)
//	}
//	// adapter is ready — credential-resolved and verifier-warm.
package clientcache
