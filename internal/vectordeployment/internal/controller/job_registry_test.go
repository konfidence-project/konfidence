package controller

import (
	"context"
	"errors"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Artifact manifest job registry", func() {
	It("removes and cancels a job", func() {
		registry := newJobRegistry()
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
		registry.remove(nn)
		Eventually(canceled).Should(BeClosed())

		_, exists := registry.get(nn)
		Expect(exists).To(BeFalse())
	})

	It("clears retry state when a job succeeds or is removed", func() {
		registry := newJobRegistry()
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

		registry.remove(nn)
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

	It("shuts down all jobs and rejects later launches", func() {
		registry := newJobRegistry()

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
		registry := newJobRegistry()
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
})
