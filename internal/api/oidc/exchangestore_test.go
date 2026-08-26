package oidc

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ExchangeCacheStore", func() {
	var (
		ctx   context.Context
		store *ExchangeCacheStore
	)

	BeforeEach(func() {
		ctx = context.Background()
		store = NewExchangeCacheStore(time.Minute)
	})

	Describe("Save", func() {
		It("rejects an empty exchange code", func() {
			err := store.Save(ctx, "", Exchange{
				SessionID:     "session-id",
				CodeChallenge: "challenge",
			})

			Expect(err).To(MatchError("exchange code must not be empty"))
		})

		It("rejects an empty session ID", func() {
			err := store.Save(ctx, "exchange-code", Exchange{
				CodeChallenge: "challenge",
			})

			Expect(err).To(MatchError("exchange session ID must not be empty"))
		})

		It("rejects an empty code challenge", func() {
			err := store.Save(ctx, "exchange-code", Exchange{
				SessionID: "session-id",
			})

			Expect(err).To(MatchError("exchange code challenge must not be empty"))
		})
	})

	Describe("Consume", func() {
		It("returns and removes a stored exchange", func() {
			expected := Exchange{
				SessionID:     "session-id",
				CodeChallenge: "challenge",
			}

			Expect(store.Save(ctx, "exchange-code", expected)).To(Succeed())

			exchange, err := store.Consume(ctx, "exchange-code")
			Expect(err).NotTo(HaveOccurred())
			Expect(exchange).To(Equal(&expected))

			exchange, err = store.Consume(ctx, "exchange-code")
			Expect(err).NotTo(HaveOccurred())
			Expect(exchange).To(BeNil())
		})

		It("returns nil for an unknown or empty code", func() {
			exchange, err := store.Consume(ctx, "")
			Expect(err).NotTo(HaveOccurred())
			Expect(exchange).To(BeNil())

			exchange, err = store.Consume(ctx, "unknown")
			Expect(err).NotTo(HaveOccurred())
			Expect(exchange).To(BeNil())
		})

		It("allows only one concurrent consumer", func() {
			Expect(store.Save(ctx, "exchange-code", Exchange{
				SessionID:     "session-id",
				CodeChallenge: "challenge",
			})).To(Succeed())

			var successfulConsumers atomic.Int32
			var waitGroup sync.WaitGroup

			for range 20 {
				waitGroup.Add(1)
				go func() {
					defer GinkgoRecover()
					defer waitGroup.Done()

					exchange, err := store.Consume(ctx, "exchange-code")
					Expect(err).NotTo(HaveOccurred())
					if exchange != nil {
						successfulConsumers.Add(1)
					}
				}()
			}

			waitGroup.Wait()
			Expect(successfulConsumers.Load()).To(Equal(int32(1)))
		})

		It("expires exchanges", func() {
			store = NewExchangeCacheStore(20 * time.Millisecond)
			Expect(store.Save(ctx, "exchange-code", Exchange{
				SessionID:     "session-id",
				CodeChallenge: "challenge",
			})).To(Succeed())

			Eventually(func() *Exchange {
				exchange, err := store.Consume(ctx, "exchange-code")
				Expect(err).NotTo(HaveOccurred())
				return exchange
			}).Should(BeNil())
		})
	})
})
