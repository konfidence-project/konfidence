package controller

import (
	"context"
	"errors"
	"sync/atomic"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/konfidence-project/konfidence/internal/vectorassembly/internal/vector"
	"github.com/konfidence-project/konfidence/pkg/observability"
	metrictest "github.com/konfidence-project/konfidence/pkg/testutil/metrics"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	"ocm.software/open-component-model/bindings/go/runtime"
)

type vectorCacheTestAdapter struct {
	calls  atomic.Int32
	result vector.Vector
	err    error
}

func (a *vectorCacheTestAdapter) GetVector(context.Context, compref.Ref) (vector.Vector, error) {
	a.calls.Add(1)
	return a.result, a.err
}

func (*vectorCacheTestAdapter) GetArtifacts(context.Context, []compref.Ref) ([]vector.Artifact, error) {
	return nil, nil
}

func (*vectorCacheTestAdapter) CreateVector(context.Context, runtime.Typed, vector.Vector) error {
	return nil
}

var _ = Describe("Vector cache metrics", func() {
	newReconciler := func() (*VectorTemplateReconciler, *prometheus.Registry) {
		cache, err := lru.New[string, vector.Vector](VectorCacheSize)
		Expect(err).NotTo(HaveOccurred())

		registry := prometheus.NewPedanticRegistry()
		metrics, err := observability.New(registry)
		Expect(err).NotTo(HaveOccurred())

		return &VectorTemplateReconciler{
			VectorCache: cache,
			Metrics:     metrics,
		}, registry
	}

	metricValue := func(registry prometheus.Gatherer, result string) float64 {
		value, err := metrictest.Value(registry, "konfidence_vectorassembly_vector_cache_lookups_total",
			prometheus.Labels{"result": result},
		)
		Expect(err).NotTo(HaveOccurred())
		return value
	}

	It("records a miss followed by a hit", func() {
		r, registry := newReconciler()
		adapter := &vectorCacheTestAdapter{
			result: vector.Vector{
				Name:    "example.com/vector",
				Version: "1.0.0",
			},
		}
		ref := compref.Ref{
			Component: "example.com/vector",
			Version:   "1.0.0",
		}

		_, err := r.getVectorCached(context.Background(), adapter, ref)
		Expect(err).NotTo(HaveOccurred())

		_, err = r.getVectorCached(context.Background(), adapter, ref)
		Expect(err).NotTo(HaveOccurred())

		Expect(adapter.calls.Load()).To(Equal(int32(1)))
		Expect(metricValue(registry, observability.CacheResultMiss)).To(Equal(float64(1)))
		Expect(metricValue(registry, observability.CacheResultHit)).To(Equal(float64(1)))
	})

	It("records failed fetches as misses without caching them", func() {
		r, registry := newReconciler()
		adapter := &vectorCacheTestAdapter{
			err: errors.New("registry unavailable"),
		}
		ref := compref.Ref{
			Component: "example.com/vector",
			Version:   "1.0.0",
		}

		_, err := r.getVectorCached(context.Background(), adapter, ref)
		Expect(err).To(HaveOccurred())

		_, err = r.getVectorCached(context.Background(), adapter, ref)
		Expect(err).To(HaveOccurred())

		Expect(adapter.calls.Load()).To(Equal(int32(2)))
		Expect(metricValue(registry, observability.CacheResultMiss)).To(Equal(float64(2)))
		Expect(metricValue(registry, observability.CacheResultHit)).To(BeZero())
	})
})
