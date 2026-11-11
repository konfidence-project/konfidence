package controller

import (
	"context"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
)

const testUID types.UID = "uid"

var _ = Describe("jobRegistry", func() {
	var (
		reg *jobRegistry
		nn  types.NamespacedName
	)

	BeforeEach(func() {
		reg = newJobRegistry()
		nn = types.NamespacedName{
			Namespace: "default",
			Name:      "test",
		}
		DeferCleanup(reg.shutdown)
	})

	It("returns no job before any launch", func() {
		_, exists := reg.get(nn)
		Expect(exists).To(BeFalse())
	})

	It("returns false when a running job has no result", func() {
		started := make(chan struct{})
		release := make(chan struct{})

		launched := reg.launch(nn, 1, testUID, func(ctx context.Context) assemblyResult {
			close(started)

			select {
			case <-release:
				return assemblyResult{vectorVersion: "1.0.0"}
			case <-ctx.Done():
				return assemblyResult{err: ctx.Err()}
			}
		})
		Expect(launched).To(BeTrue())
		Eventually(started).Should(BeClosed())

		job, exists := reg.get(nn)
		Expect(exists).To(BeTrue())

		_, completed := reg.completedResult(nn, job)
		Expect(completed).To(BeFalse())

		close(release)

		var result assemblyResult
		Eventually(func() bool {
			result, completed = reg.completedResult(nn, job)
			return completed
		}).Should(BeTrue())

		Expect(result.err).NotTo(HaveOccurred())
		Expect(result.vectorVersion).To(Equal("1.0.0"))
	})

	It("atomically consumes and removes a completed result", func() {
		launched := reg.launch(nn, 1, testUID, func(context.Context) assemblyResult {
			return assemblyResult{vectorVersion: "1.0.0"}
		})
		Expect(launched).To(BeTrue())

		job, exists := reg.get(nn)
		Expect(exists).To(BeTrue())
		Expect(job.generation).To(Equal(int64(1)))

		var (
			result    assemblyResult
			completed bool
		)
		Eventually(func() bool {
			result, completed = reg.completedResult(nn, job)
			return completed
		}).Should(BeTrue())

		Expect(result.vectorVersion).To(Equal("1.0.0"))

		_, exists = reg.get(nn)
		Expect(exists).To(BeFalse())

		_, completed = reg.completedResult(nn, job)
		Expect(completed).To(BeFalse())
	})

	It("removes and cancels a running job", func() {
		started := make(chan struct{})
		canceled := make(chan struct{})

		launched := reg.launch(nn, 1, testUID, func(ctx context.Context) assemblyResult {
			close(started)
			<-ctx.Done()
			close(canceled)
			return assemblyResult{err: ctx.Err()}
		})
		Expect(launched).To(BeTrue())
		Eventually(started).Should(BeClosed())

		reg.remove(nn)

		Eventually(canceled).Should(BeClosed())

		_, exists := reg.get(nn)
		Expect(exists).To(BeFalse())
	})

	It("remove only affects the targeted key", func() {
		other := types.NamespacedName{
			Namespace: "default",
			Name:      "other",
		}

		firstCanceled := make(chan struct{})
		secondCanceled := make(chan struct{})

		Expect(reg.launch(nn, 1, testUID, func(ctx context.Context) assemblyResult {
			<-ctx.Done()
			close(firstCanceled)
			return assemblyResult{err: ctx.Err()}
		})).To(BeTrue())

		Expect(reg.launch(other, 2, types.UID("other-uid"), func(ctx context.Context) assemblyResult {
			<-ctx.Done()
			close(secondCanceled)
			return assemblyResult{err: ctx.Err()}
		})).To(BeTrue())

		reg.remove(nn)

		Eventually(firstCanceled).Should(BeClosed())
		Consistently(secondCanceled).ShouldNot(BeClosed())

		_, exists := reg.get(nn)
		Expect(exists).To(BeFalse())

		otherJob, exists := reg.get(other)
		Expect(exists).To(BeTrue())
		Expect(otherJob.generation).To(Equal(int64(2)))
	})

	It("cancels an existing job before replacing it", func() {
		const (
			oldUID types.UID = "old-uid"
			newUID types.UID = "new-uid"
		)

		firstStarted := make(chan struct{})
		firstCanceled := make(chan struct{})

		Expect(reg.launch(nn, 1, oldUID, func(ctx context.Context) assemblyResult {
			close(firstStarted)
			<-ctx.Done()
			close(firstCanceled)
			return assemblyResult{err: ctx.Err()}
		})).To(BeTrue())

		Eventually(firstStarted).Should(BeClosed())

		oldJob, exists := reg.get(nn)
		Expect(exists).To(BeTrue())
		Expect(oldJob.matches(1, oldUID)).To(BeTrue())

		Expect(reg.launch(nn, 1, newUID, func(context.Context) assemblyResult {
			return assemblyResult{vectorVersion: "2.0.0"}
		})).To(BeTrue())

		Eventually(firstCanceled).Should(BeClosed())

		replacement, exists := reg.get(nn)
		Expect(exists).To(BeTrue())
		Expect(replacement).NotTo(BeIdenticalTo(oldJob))
		Expect(replacement.matches(1, newUID)).To(BeTrue())
		Expect(replacement.matches(1, oldUID)).To(BeFalse())

		By("rejecting the replaced job's result")
		_, completed := reg.completedResult(nn, oldJob)
		Expect(completed).To(BeFalse())

		By("preserving the replacement job's result")
		var result assemblyResult
		Eventually(func() bool {
			result, completed = reg.completedResult(nn, replacement)
			return completed
		}).Should(BeTrue())

		Expect(result.err).NotTo(HaveOccurred())
		Expect(result.vectorVersion).To(Equal("2.0.0"))
	})

	It("shuts down all jobs and rejects later launches", func() {
		started := make(chan struct{}, 2)
		exited := make(chan struct{}, 2)

		for _, name := range []string{"first", "second"} {
			key := types.NamespacedName{
				Namespace: "default",
				Name:      name,
			}

			Expect(reg.launch(key, 1, types.UID(name), func(ctx context.Context) assemblyResult {
				started <- struct{}{}
				<-ctx.Done()
				exited <- struct{}{}
				return assemblyResult{err: ctx.Err()}
			})).To(BeTrue())
		}

		Eventually(started).Should(Receive())
		Eventually(started).Should(Receive())

		shutdownDone := make(chan struct{})
		go func() {
			reg.shutdown()
			close(shutdownDone)
		}()

		Eventually(exited).Should(Receive())
		Eventually(exited).Should(Receive())
		Eventually(shutdownDone).Should(BeClosed())

		Expect(reg.inflight).To(BeEmpty())

		launched := reg.launch(
			types.NamespacedName{Namespace: "default", Name: "late"},
			1,
			types.UID("late"),
			func(context.Context) assemblyResult {
				return assemblyResult{}
			},
		)
		Expect(launched).To(BeFalse())

		By("allowing shutdown to be called repeatedly")
		reg.shutdown()
	})

	DescribeTable("matches the Kubernetes object identity",
		func(
			jobGeneration int64,
			jobUID types.UID,
			currentGeneration int64,
			currentUID types.UID,
			expected bool,
		) {
			identity := assemblyJobIdentity{
				generation: jobGeneration,
				uid:        jobUID,
			}

			Expect(identity.matches(currentGeneration, currentUID)).To(Equal(expected))
		},
		Entry(
			"when generation and UID match",
			int64(1), types.UID("uid-1"),
			int64(1), types.UID("uid-1"),
			true,
		),
		Entry(
			"when generation changes",
			int64(1), types.UID("uid-1"),
			int64(2), types.UID("uid-1"),
			false,
		),
		Entry(
			"when UID changes",
			int64(1), types.UID("uid-old"),
			int64(1), types.UID("uid-new"),
			false,
		),
	)

	It("publishes a failed result when an assembly job panics", func() {
		launched := reg.launch(nn, 1, testUID, func(context.Context) assemblyResult {
			return runAssemblyJob(logr.Discard(), func() assemblyResult {
				panic("unexpected failure")
			})
		})
		Expect(launched).To(BeTrue())

		job, exists := reg.get(nn)
		Expect(exists).To(BeTrue())

		var (
			result    assemblyResult
			completed bool
		)
		Eventually(func() bool {
			result, completed = reg.completedResult(nn, job)
			return completed
		}).Should(BeTrue())

		Expect(result.err).To(MatchError("assembly job panicked: unexpected failure"))

		By("removing the completed job from the registry")
		_, exists = reg.get(nn)
		Expect(exists).To(BeFalse())
	})
})
