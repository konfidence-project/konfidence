package oidc

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("StateCacheStore", func() {
	var (
		ctx   context.Context
		store *StateCacheStore
	)

	BeforeEach(func() {
		ctx = context.Background()
		store = NewStateCacheStore(time.Minute)
	})

	It("rejects nil state", func() {
		Expect(store.Save(ctx, nil)).To(
			MatchError("failed to store state: state is empty"),
		)
	})

	It("returns and removes a stored state", func() {
		expected := &StateData{
			State:               "state-id",
			Nonce:               "nonce",
			ReturnURL:           "http://127.0.0.1:1234/callback",
			ClientCodeChallenge: "challenge",
		}

		Expect(store.Save(ctx, nil)).To(
			MatchError("failed to store state: state is empty"),
		)
		Expect(store.Save(ctx, expected)).To(Succeed())
		actual, err := store.Consume(ctx, "state-id")
		Expect(err).NotTo(HaveOccurred())
		Expect(actual).To(Equal(expected))

		actual, err = store.Consume(ctx, "state-id")
		Expect(err).NotTo(HaveOccurred())
		Expect(actual).To(BeNil())
	})

	It("allows only one concurrent consumer", func() {
		Expect(store.Save(ctx, &StateData{State: "state-id"})).To(Succeed())

		var successfulConsumers atomic.Int32
		var waitGroup sync.WaitGroup

		for range 20 {
			waitGroup.Add(1)
			go func() {
				defer GinkgoRecover()
				defer waitGroup.Done()

				state, err := store.Consume(ctx, "state-id")
				Expect(err).NotTo(HaveOccurred())
				if state != nil {
					successfulConsumers.Add(1)
				}
			}()
		}

		waitGroup.Wait()
		Expect(successfulConsumers.Load()).To(Equal(int32(1)))
	})
})
