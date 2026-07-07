package crypto

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Limiter", func() {
	Describe("NoopLimiter", func() {
		It("acquires immediately and release is a no-op", func() {
			l := NoopLimiter{}
			release, err := l.Acquire(context.Background())
			Expect(err).ToNot(HaveOccurred())
			Expect(release).ToNot(BeNil())
			release()
			// idempotent calls would be undefined for the real limiter, but
			// NoopLimiter is intentionally cheap — calling Acquire many times
			// must never block.
			for i := 0; i < 8; i++ {
				r, err := l.Acquire(context.Background())
				Expect(err).ToNot(HaveOccurred())
				r()
			}
		})

		It("returns ctx.Err() when the context is already cancelled", func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			release, err := NoopLimiter{}.Acquire(ctx)
			Expect(err).To(MatchError(context.Canceled))
			Expect(release).To(BeNil())
		})
	})

	Describe("NewLimiter", func() {
		It("returns a non-nil limiter for n > 0", func() {
			Expect(NewLimiter(1)).ToNot(BeNil())
		})

		It("falls back to GOMAXPROCS when n <= 0", func() {
			// observable proxy: a limiter built with n=0 must allow at least
			// GOMAXPROCS concurrent holders without blocking.
			cap := runtime.GOMAXPROCS(0)
			l := NewLimiter(0)
			releases := make([]func(), cap)
			for i := 0; i < cap; i++ {
				r, err := l.Acquire(context.Background())
				Expect(err).ToNot(HaveOccurred())
				releases[i] = r
			}
			for _, r := range releases {
				r()
			}
		})

		It("bounds concurrent holders to the configured cap", func() {
			const cap = 3
			const goroutines = 12
			const hold = 50 * time.Millisecond

			l := NewLimiter(cap)
			var inflight, peak atomic.Int32
			var wg sync.WaitGroup
			wg.Add(goroutines)
			for i := 0; i < goroutines; i++ {
				go func() {
					defer wg.Done()
					release, err := l.Acquire(context.Background())
					Expect(err).ToNot(HaveOccurred())
					defer release()
					cur := inflight.Add(1)
					for {
						p := peak.Load()
						if cur <= p || peak.CompareAndSwap(p, cur) {
							break
						}
					}
					time.Sleep(hold)
					inflight.Add(-1)
				}()
			}
			wg.Wait()
			Expect(int(peak.Load())).To(BeNumerically("<=", cap))
			Expect(int(peak.Load())).To(BeNumerically(">=", 1))
		})

		It("respects context cancellation while waiting for a slot", func() {
			l := NewLimiter(1)
			holdRelease, err := l.Acquire(context.Background())
			Expect(err).ToNot(HaveOccurred())
			defer holdRelease()

			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
			defer cancel()

			start := time.Now()
			release, err := l.Acquire(ctx)
			elapsed := time.Since(start)

			Expect(err).To(HaveOccurred())
			Expect(release).To(BeNil())
			// the context deadline must drive the failure, not a longer wait
			Expect(elapsed).To(BeNumerically("<", 200*time.Millisecond))
		})

		It("returns a slot to the pool after release", func() {
			l := NewLimiter(1)
			r1, err := l.Acquire(context.Background())
			Expect(err).ToNot(HaveOccurred())
			r1()

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			r2, err := l.Acquire(ctx)
			Expect(err).ToNot(HaveOccurred())
			r2()
		})
	})
})
