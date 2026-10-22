package cmd

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/konfidence-project/konfidence/internal/api/server"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func informerClientFixture() *informerBackedClient {
	return &informerBackedClient{
		cache: &informerBackedCache{
			done: make(chan struct{}),
		},
	}
}

func stopInformer(informer *informerBackedClient, err error) {
	informer.cache.runErr = err
	close(informer.cache.done)
}

var _ = Describe("Informer lifecycle", func() {
	It("reports an unexpected terminal error", func() {
		informer := informerClientFixture()
		expected := errors.New("cache failed")
		stopInformer(informer, expected)

		err := waitForInformerTermination(context.Background(), informer)
		Expect(err).To(MatchError(ContainSubstring("informer cache stopped unexpectedly: cache failed")))
	})

	It("reports unexpected termination without an error", func() {
		informer := informerClientFixture()
		stopInformer(informer, nil)

		err := waitForInformerTermination(context.Background(), informer)
		Expect(err).To(MatchError("informer cache stopped unexpectedly"))
	})

	It("returns cleanly when the application context is canceled", func() {
		informer := informerClientFixture()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := waitForInformerTermination(ctx, informer)
		Expect(err).NotTo(HaveOccurred())
	})

	It("does not report informer termination during normal shutdown", func() {
		informer := informerClientFixture()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		stopInformer(informer, context.Canceled)
		err := waitForInformerTermination(ctx, informer)
		Expect(err).NotTo(HaveOccurred())
	})

	It("allows multiple consumers to observe the terminal result", func() {
		informer := informerClientFixture()
		expected := errors.New("cache failed")
		stopInformer(informer, expected)

		results := make(chan error, 2)
		for range 2 {
			go func() {
				results <- informer.Wait()
			}()
		}

		Eventually(results).Should(Receive(MatchError(expected)))
		Eventually(results).Should(Receive(MatchError(expected)))
	})

	It("shuts down the HTTP server when the informer stops", func() {
		informer := informerClientFixture()
		informerErr := errors.New("cache failed")

		cfg := config.Parsed{
			Server: config.ParsedServerConfig{
				Addr:            "127.0.0.1:0",
				ReadTimeout:     time.Second,
				WriteTimeout:    time.Second,
				ShutdownTimeout: time.Second,
			},
		}

		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		apiServer := server.New(cfg, logger, http.NotFoundHandler())

		ctx, cancel := context.WithCancel(context.Background())
		DeferCleanup(cancel)

		addrCh := make(chan string, 1)
		errCh := make(chan error, 1)

		go func() {
			errCh <- runServerWithInformer(ctx, apiServer, informer,
				func(addr string) {
					addrCh <- addr
				},
			)
		}()

		var addr string
		Eventually(addrCh, 3*time.Second).Should(Receive(&addr))

		resp, err := http.Get("http://" + addr + "/healthz") //nolint:noctx
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		Expect(resp.Body.Close()).To(Succeed())

		stopInformer(informer, informerErr)

		var runErr error
		Eventually(errCh, 3*time.Second).Should(Receive(&runErr))
		Expect(runErr).To(MatchError(ContainSubstring("informer cache stopped unexpectedly: cache failed")))

		Eventually(func() error {
			conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
			if err != nil {
				return nil
			}

			_ = conn.Close()
			return errors.New("HTTP server still accepts connections")
		}, 3*time.Second, 50*time.Millisecond).Should(Succeed())
	})
})
