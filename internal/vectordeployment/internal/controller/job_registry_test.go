package controller

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/konfidence-project/konfidence/pkg/observability"
	metrictest "github.com/konfidence-project/konfidence/pkg/testutil/metrics"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Artifact manifest job registry", func() {
	It("removes and cancels a job", func() {
		registry := newJobRegistry(nil)
		nn := types.NamespacedName{Namespace: "default", Name: "vector"}

		started := make(chan struct{})
		canceled := make(chan struct{})

		launched, retryAfter := registry.launch(nn, 1, types.UID("uid"), []string{"ref"},

			func(ctx context.Context) artifactManifestResult {
				close(started)
				<-ctx.Done()
				close(canceled)
				return artifactManifestResult{err: ctx.Err()}
			},
		)
		Expect(launched).To(BeTrue())
		Expect(retryAfter).To(BeZero())

		Eventually(started).Should(BeClosed())
		Expect(registry.remove(nn)).To(BeTrue())
		Eventually(canceled).Should(BeClosed())

		_, exists := registry.get(nn)
		Expect(exists).To(BeFalse())
	})

	It("clears retry state when a job succeeds or is removed", func() {
		registry := newJobRegistry(nil)
		nn := types.NamespacedName{Namespace: "default", Name: "vector"}

		launched, retryAfter := registry.launch(nn, 1, types.UID("uid"), []string{"ref"},
			func(context.Context) artifactManifestResult {
				return artifactManifestResult{err: errors.New("registry unavailable")}
			},
		)
		Expect(launched).To(BeTrue())
		Expect(retryAfter).To(BeZero())

		job, exists := registry.get(nn)
		Expect(exists).To(BeTrue())

		var result artifactManifestResult
		Eventually(func() bool {
			var completed bool
			result, completed = registry.completedResult(nn, job)
			return completed
		}).Should(BeTrue())

		Expect(result.err).To(MatchError("registry unavailable"))
		Expect(registry.retryLimiter.NumRequeues(nn)).To(Equal(1))

		Expect(registry.remove(nn)).To(BeTrue())
		Expect(registry.retryLimiter.NumRequeues(nn)).To(BeZero())

		launched, retryAfter = registry.launch(nn, 1, types.UID("uid"), []string{"ref"},
			func(context.Context) artifactManifestResult {
				return artifactManifestResult{}
			},
		)
		Expect(launched).To(BeTrue())
		Expect(retryAfter).To(BeZero())

		job, exists = registry.get(nn)
		Expect(exists).To(BeTrue())

		Eventually(func() bool {
			var completed bool
			result, completed = registry.completedResult(nn, job)
			return completed
		}).Should(BeTrue())

		Expect(result.err).NotTo(HaveOccurred())
		Expect(registry.retryLimiter.NumRequeues(nn)).To(BeZero())

		By("retaining the successful result in the registry")
		cachedJob, exists := registry.get(nn)
		Expect(exists).To(BeTrue())
		Expect(cachedJob).To(BeIdenticalTo(job))

		cachedResult, completed := registry.completedResult(nn, cachedJob)
		Expect(completed).To(BeTrue())
		Expect(cachedResult.err).NotTo(HaveOccurred())
		Expect(registry.retryLimiter.NumRequeues(nn)).To(BeZero())
	})

	It("caches a permanent result until the job is removed", func() {
		registry := newJobRegistry(nil)
		DeferCleanup(registry.shutdown)

		nn := types.NamespacedName{
			Namespace: "default",
			Name:      "vector",
		}
		permanent := &PermanentError{
			Reason: "ManifestMissing",
			Err:    errors.New("manifest missing"),
		}

		launched, retryAfter := registry.launch(nn, 1, types.UID("uid"), []string{"ref"},
			func(context.Context) artifactManifestResult {
				return artifactManifestResult{err: permanent}
			},
		)
		Expect(launched).To(BeTrue())
		Expect(retryAfter).To(BeZero())

		job, exists := registry.get(nn)
		Expect(exists).To(BeTrue())

		var (
			result    artifactManifestResult
			completed bool
		)
		Eventually(func() bool {
			result, completed = registry.completedResult(nn, job)
			return completed
		}).Should(BeTrue())

		var actual *PermanentError
		Expect(errors.As(result.err, &actual)).To(BeTrue())
		Expect(actual).To(BeIdenticalTo(permanent))
		Expect(registry.retryLimiter.NumRequeues(nn)).To(BeZero())

		By("retaining the permanent result in the registry")
		cachedJob, exists := registry.get(nn)
		Expect(exists).To(BeTrue())
		Expect(cachedJob).To(BeIdenticalTo(job))
		Expect(cachedJob.cachedResult).NotTo(BeNil())

		By("returning the cached error without entering retry backoff")
		cachedResult, completed := registry.completedResult(nn, cachedJob)
		Expect(completed).To(BeTrue())
		Expect(errors.As(cachedResult.err, &actual)).To(BeTrue())
		Expect(actual).To(BeIdenticalTo(permanent))
		Expect(registry.retryLimiter.NumRequeues(nn)).To(BeZero())

		By("removing the cached result")
		Expect(registry.remove(nn)).To(BeTrue())

		_, exists = registry.get(nn)
		Expect(exists).To(BeFalse())

		_, completed = registry.completedResult(nn, cachedJob)
		Expect(completed).To(BeFalse())
		Expect(registry.retryLimiter.NumRequeues(nn)).To(BeZero())
	})

	It("shuts down all jobs and rejects later launches", func() {
		registry := newJobRegistry(nil)

		var running atomic.Int32
		exited := make(chan struct{}, 2)

		for _, name := range []string{"a", "b"} {
			nn := types.NamespacedName{Namespace: "default", Name: name}
			launched, retryAfter := registry.launch(nn, 1, types.UID(name), []string{name},
				func(ctx context.Context) artifactManifestResult {
					running.Add(1)
					<-ctx.Done()
					running.Add(-1)
					exited <- struct{}{}
					return artifactManifestResult{err: ctx.Err()}
				},
			)
			Expect(launched).To(BeTrue())
			Expect(retryAfter).To(BeZero())
		}

		Eventually(running.Load).Should(Equal(int32(2)))

		shutdownDone := make(chan struct{})
		go func() {
			registry.shutdown()
			close(shutdownDone)
		}()

		Eventually(exited).Should(Receive())
		Eventually(exited).Should(Receive())
		Eventually(shutdownDone).Should(BeClosed())
		Expect(running.Load()).To(Equal(int32(0)))

		Expect(registry.inflight).To(BeEmpty())

		launched, retryAfter := registry.launch(types.NamespacedName{Name: "late"}, 1, types.UID("late"), nil,
			func(context.Context) artifactManifestResult {
				return artifactManifestResult{}
			},
		)
		Expect(launched).To(BeFalse())
		Expect(retryAfter).To(BeZero())

		By("allowing repeated shutdown")
		registry.shutdown()
	})

	It("cancels an existing job before replacing it", func() {
		registry := newJobRegistry(nil)
		DeferCleanup(registry.shutdown)

		nn := types.NamespacedName{
			Namespace: "default",
			Name:      "vector",
		}

		started := make(chan struct{})
		canceled := make(chan struct{})

		launched, _ := registry.launch(nn, 1, types.UID("uid"), []string{"old-ref"},
			func(ctx context.Context) artifactManifestResult {
				close(started)
				<-ctx.Done()
				close(canceled)
				return artifactManifestResult{err: ctx.Err()}
			},
		)
		Expect(launched).To(BeTrue())
		Eventually(started).Should(BeClosed())

		oldJob, exists := registry.get(nn)
		Expect(exists).To(BeTrue())

		launched, _ = registry.launch(nn, 2, types.UID("uid"), []string{"new-ref"},
			func(context.Context) artifactManifestResult {
				return artifactManifestResult{}
			},
		)
		Expect(launched).To(BeTrue())
		Eventually(canceled).Should(BeClosed())

		replacement, exists := registry.get(nn)
		Expect(exists).To(BeTrue())
		Expect(replacement).NotTo(BeIdenticalTo(oldJob))
		Expect(replacement.matches(2, types.UID("uid"), []string{"new-ref"})).To(BeTrue())
		By("rejecting results from the replaced job")
		_, completed := registry.completedResult(nn, oldJob)
		Expect(completed).To(BeFalse())

		By("preserving the replacement result")
		var result artifactManifestResult
		Eventually(func() bool {
			var completed bool
			result, completed = registry.completedResult(nn, replacement)
			return completed
		}).Should(BeTrue())

		Expect(result.err).NotTo(HaveOccurred())
	})

	It("records misses until a completed result is cached, then records a hit", func() {
		promRegistry := prometheus.NewPedanticRegistry()
		metrics, err := observability.New(promRegistry)
		Expect(err).NotTo(HaveOccurred())

		registry := newJobRegistry(metrics)
		DeferCleanup(registry.shutdown)

		nn := types.NamespacedName{
			Namespace: "default",
			Name:      "vector",
		}

		release := make(chan struct{})
		launched, retryAfter := registry.launch(nn, 1, types.UID("uid"), []string{"ref"},
			func(context.Context) artifactManifestResult {
				<-release
				return artifactManifestResult{}
			},
		)
		Expect(launched).To(BeTrue())
		Expect(retryAfter).To(BeZero())

		job, exists := registry.get(nn)
		Expect(exists).To(BeTrue())

		_, completed := registry.completedResult(nn, job)
		Expect(completed).To(BeFalse())

		close(release)
		Eventually(func() int {
			return len(job.result)
		}).Should(Equal(1))

		result, completed := registry.completedResult(nn, job)
		Expect(completed).To(BeTrue())
		Expect(result.err).NotTo(HaveOccurred())

		result, completed = registry.completedResult(nn, job)
		Expect(completed).To(BeTrue())
		Expect(result.err).NotTo(HaveOccurred())

		misses, err := metrictest.Value(promRegistry, "konfidence_vectordeployment_manifest_cache_lookups_total",
			prometheus.Labels{
				"result": observability.CacheResultMiss,
			},
		)
		Expect(err).NotTo(HaveOccurred())

		// Initial launch, running poll, and first completion consumption are all
		// lookups without an already-cached result.
		Expect(misses).To(Equal(float64(3)))

		hits, err := metrictest.Value(promRegistry, "konfidence_vectordeployment_manifest_cache_lookups_total",
			prometheus.Labels{
				"result": observability.CacheResultHit,
			},
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(hits).To(Equal(float64(1)))
	})

	It("records backoff as a miss and increments retry only when the retry launches", func() {
		promRegistry := prometheus.NewPedanticRegistry()
		metrics, err := observability.New(promRegistry)
		Expect(err).NotTo(HaveOccurred())

		registry := newJobRegistry(metrics)
		DeferCleanup(registry.shutdown)

		now := time.Date(2026, time.November, 25, 12, 0, 0, 0, time.UTC)
		registry.now = func() time.Time { return now }

		nn := types.NamespacedName{
			Namespace: "default",
			Name:      "vector",
		}

		launched, _ := registry.launch(nn, 1, types.UID("uid"), []string{"ref"},
			func(context.Context) artifactManifestResult {
				return artifactManifestResult{
					err: errors.New("registry unavailable"),
				}
			},
		)
		Expect(launched).To(BeTrue())

		job, exists := registry.get(nn)
		Expect(exists).To(BeTrue())

		Eventually(func() int {
			return len(job.result)
		}).Should(Equal(1))

		result, completed := registry.completedResult(nn, job)
		Expect(completed).To(BeTrue())
		Expect(result.err).To(HaveOccurred())

		launched, retryAfter := registry.launch(nn, 1, types.UID("uid"), []string{"ref"},
			func(context.Context) artifactManifestResult {
				return artifactManifestResult{}
			},
		)
		Expect(launched).To(BeFalse())
		Expect(retryAfter).To(Equal(defaultArtifactManifestRetryBaseDelay))

		retries, err := metrictest.Value(promRegistry, "konfidence_vectordeployment_manifest_job_retries_total", nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(retries).To(BeZero())

		now = now.Add(defaultArtifactManifestRetryBaseDelay)

		launched, retryAfter = registry.launch(nn, 1, types.UID("uid"), []string{"ref"},
			func(context.Context) artifactManifestResult {
				return artifactManifestResult{}
			},
		)
		Expect(launched).To(BeTrue())
		Expect(retryAfter).To(BeZero())

		retries, err = metrictest.Value(promRegistry, "konfidence_vectordeployment_manifest_job_retries_total", nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(retries).To(Equal(float64(1)))

		misses, err := metrictest.Value(promRegistry, "konfidence_vectordeployment_manifest_cache_lookups_total",
			prometheus.Labels{
				"result": observability.CacheResultMiss,
			},
		)
		Expect(err).NotTo(HaveOccurred())

		// Initial launch, transient-result consumption, backoff poll, and retry
		// launch are all lookups without an already-cached result.
		Expect(misses).To(Equal(float64(4)))
	})
	DescribeTable(
		"records identity invalidation",
		func(oldGeneration, newGeneration int64, oldUID, newUID types.UID, oldReferences, newReferences []string, expectedReason string) {
			promRegistry := prometheus.NewPedanticRegistry()
			metrics, err := observability.New(promRegistry)
			Expect(err).NotTo(HaveOccurred())

			registry := newJobRegistry(metrics)
			DeferCleanup(registry.shutdown)

			nn := types.NamespacedName{
				Namespace: "default",
				Name:      "vector",
			}

			registry.retries[nn] = &manifestRetry{
				manifestJobIdentity: newManifestJobIdentity(oldGeneration, oldUID, oldReferences),
				notBefore:           time.Now().Add(time.Hour),
			}

			launched, _ := registry.launch(nn, newGeneration, newUID, newReferences,
				func(context.Context) artifactManifestResult {
					return artifactManifestResult{}
				},
			)
			Expect(launched).To(BeTrue())

			value, err := metrictest.Value(promRegistry, "konfidence_vectordeployment_manifest_cache_invalidations_total",
				prometheus.Labels{"reason": expectedReason},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(value).To(Equal(float64(1)))
		},
		Entry("for a UID change", int64(1), int64(1), types.UID("old"), types.UID("new"), []string{"ref"}, []string{"ref"},
			observability.ManifestInvalidationUIDChanged,
		),
		Entry("for a generation change", int64(1), int64(2), types.UID("uid"), types.UID("uid"), []string{"ref"}, []string{"ref"},
			observability.ManifestInvalidationGenerationChanged,
		),
		Entry("for a reference change", int64(1), int64(1), types.UID("uid"), types.UID("uid"), []string{"old"}, []string{"new"},
			observability.ManifestInvalidationReferencesChanged,
		),
	)

})
