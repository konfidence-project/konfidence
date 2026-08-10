package lrucache_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/konfidence-project/konfidence/pkg/lrucache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// cr is a minimal stand-in for a reconciled CR.
type cr struct {
	namespace  string
	name       string
	generation int64
}

func extract(c cr) uint64 {
	return lrucache.HashKey(c.namespace, c.name, c.generation)
}

var _ = Describe("Cache", func() {
	var (
		calls atomic.Int32
		cache *lrucache.Cache[cr, string]
	)

	BeforeEach(func() {
		calls.Store(0)
		var err error
		cache, err = lrucache.New(lrucache.DefaultCacheSize, extract,
			func(_ context.Context, _ client.Reader, c cr) (string, error) {
				calls.Add(1)
				return fmt.Sprintf("%s/%s@%d", c.namespace, c.name, c.generation), nil
			},
		)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("Lookup", func() {
		It("calls the factory on a miss and returns the value", func() {
			c := cr{namespace: "ns", name: "foo", generation: 1}
			v, err := cache.Lookup(context.Background(), nil, c)
			Expect(err).NotTo(HaveOccurred())
			Expect(v).To(Equal("ns/foo@1"))
			Expect(calls.Load()).To(Equal(int32(1)))
		})

		It("returns cached value on a hit without calling the factory again", func() {
			c := cr{namespace: "ns", name: "foo", generation: 1}
			_, _ = cache.Lookup(context.Background(), nil, c)
			v, err := cache.Lookup(context.Background(), nil, c)
			Expect(err).NotTo(HaveOccurred())
			Expect(v).To(Equal("ns/foo@1"))
			Expect(calls.Load()).To(Equal(int32(1)))
		})

		It("misses when generation changes", func() {
			c1 := cr{namespace: "ns", name: "foo", generation: 1}
			c2 := cr{namespace: "ns", name: "foo", generation: 2}
			_, _ = cache.Lookup(context.Background(), nil, c1)
			v, err := cache.Lookup(context.Background(), nil, c2)
			Expect(err).NotTo(HaveOccurred())
			Expect(v).To(Equal("ns/foo@2"))
			Expect(calls.Load()).To(Equal(int32(2)))
		})

		It("propagates factory errors without caching", func() {
			boom := fmt.Errorf("factory error")
			errCache, err := lrucache.New(lrucache.DefaultCacheSize, extract,
				func(_ context.Context, _ client.Reader, _ cr) (string, error) {
					return "", boom
				},
			)
			Expect(err).NotTo(HaveOccurred())

			c := cr{namespace: "ns", name: "foo", generation: 1}
			_, err = errCache.Lookup(context.Background(), nil, c)
			Expect(err).To(MatchError(boom))

			// second call still hits the factory (error was not cached)
			_, err = errCache.Lookup(context.Background(), nil, c)
			Expect(err).To(MatchError(boom))
		})

		It("is safe for concurrent use on a cold cache", func() {
			c := cr{namespace: "ns", name: "foo", generation: 1}
			const goroutines = 50
			var wg sync.WaitGroup
			wg.Add(goroutines)
			for range goroutines {
				go func() {
					defer wg.Done()
					_, err := cache.Lookup(context.Background(), nil, c)
					Expect(err).NotTo(HaveOccurred())
				}()
			}
			wg.Wait()
			// factory may be called more than once due to concurrent misses before
			// the first result is stored, but must not panic or corrupt state.
			Expect(calls.Load()).To(BeNumerically(">=", 1))
		})

		It("calls the factory exactly once when the cache is warm", func() {
			c := cr{namespace: "ns", name: "foo", generation: 1}
			// Prime the cache.
			_, err := cache.Lookup(context.Background(), nil, c)
			Expect(err).NotTo(HaveOccurred())
			Expect(calls.Load()).To(Equal(int32(1)))

			const goroutines = 50
			var wg sync.WaitGroup
			wg.Add(goroutines)
			for range goroutines {
				go func() {
					defer wg.Done()
					_, err := cache.Lookup(context.Background(), nil, c)
					Expect(err).NotTo(HaveOccurred())
				}()
			}
			wg.Wait()
			// Cache was warm before goroutines started — factory must not be called again.
			Expect(calls.Load()).To(Equal(int32(1)))
		})
	})
})
