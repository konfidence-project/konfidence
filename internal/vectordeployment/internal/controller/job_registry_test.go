package controller

import (
	"context"
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

		Expect(registry.launch(nn, 1, types.UID("uid"), []string{"ref"},
			func(ctx context.Context) artifactManifestResult {
				close(started)
				<-ctx.Done()
				close(canceled)
				return artifactManifestResult{err: ctx.Err()}
			},
		)).To(BeTrue())

		Eventually(started).Should(BeClosed())
		registry.remove(nn)
		Eventually(canceled).Should(BeClosed())

		_, exists := registry.get(nn)
		Expect(exists).To(BeFalse())
	})

	It("shuts down all jobs and rejects later launches", func() {
		registry := newJobRegistry()

		var running atomic.Int32
		exited := make(chan struct{}, 2)

		for _, name := range []string{"a", "b"} {
			nn := types.NamespacedName{Namespace: "default", Name: name}
			Expect(registry.launch(nn, 1, types.UID(name), []string{name},
				func(ctx context.Context) artifactManifestResult {
					running.Add(1)
					<-ctx.Done()
					running.Add(-1)
					exited <- struct{}{}
					return artifactManifestResult{err: ctx.Err()}
				},
			)).To(BeTrue())
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

		Expect(registry.launch(types.NamespacedName{Name: "late"}, 1, types.UID("late"), nil,
			func(context.Context) artifactManifestResult {
				return artifactManifestResult{}
			},
		)).To(BeFalse())

		By("allowing repeated shutdown")
		registry.shutdown()
	})
})
