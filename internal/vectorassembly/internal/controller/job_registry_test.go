package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("jobRegistry", func() {
	var (
		reg *jobRegistry
		nn  types.NamespacedName
	)

	BeforeEach(func() {
		reg = newJobRegistry()
		nn = types.NamespacedName{Namespace: "default", Name: "test"}
	})

	It("returns no job before any launch", func() {
		_, ok := reg.get(nn)
		Expect(ok).To(BeFalse())
	})

	It("returns the job after launch and marks it done when the fn completes", func() {
		reg.launch(nn, 1, func(_ context.Context) assemblyResult {
			return assemblyResult{vectorVersion: "1.0.0"}
		})

		var job *inflightJob
		Eventually(func() bool {
			j, ok := reg.get(nn)
			if !ok {
				return false
			}
			job = j
			return j.done()
		}, time.Second, 10*time.Millisecond).Should(BeTrue())

		Expect(job.generation).To(Equal(int64(1)))
		res := <-job.result
		Expect(res.vectorVersion).To(Equal("1.0.0"))
	})

	It("remove cancels the context and removes the entry", func() {
		started := make(chan struct{})
		var ctxErr error

		reg.launch(nn, 1, func(ctx context.Context) assemblyResult {
			close(started)
			<-ctx.Done()
			ctxErr = ctx.Err()
			return assemblyResult{}
		})
		<-started

		reg.remove(nn)

		_, ok := reg.get(nn)
		Expect(ok).To(BeFalse())

		Eventually(func() error { return ctxErr }, time.Second, 10*time.Millisecond).
			Should(MatchError(context.Canceled))
	})

	It("remove only affects the targeted key", func() {
		nn2 := types.NamespacedName{Namespace: "default", Name: "other"}

		reg.launch(nn, 1, func(_ context.Context) assemblyResult { return assemblyResult{} })
		reg.launch(nn2, 2, func(_ context.Context) assemblyResult { return assemblyResult{vectorVersion: "gen2"} })

		reg.remove(nn)

		_, ok := reg.get(nn)
		Expect(ok).To(BeFalse())

		j2, ok := reg.get(nn2)
		Expect(ok).To(BeTrue())
		Expect(j2.generation).To(Equal(int64(2)))
	})
})
