package server_test

import (
	"context"
	"net/http"
	"time"

	"github.com/konfidence-project/konfidence/internal/api/handler"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/konfidence-project/konfidence/internal/api/server"
)

func validParsed(addr string) config.Parsed {
	cfg := config.Config{
		Addr:            addr,
		LogLevel:        "error",
		ReadTimeout:     "5s",
		WriteTimeout:    "5s",
		ShutdownTimeout: "2s",
	}
	parsed, err := cfg.Validate()
	if err != nil {
		panic(err)
	}
	return parsed
}

var _ = Describe("Server", func() {
	Describe("Run", func() {
		It("starts and stops cleanly when context is cancelled", func() {
			srv := server.New(validParsed("127.0.0.1:0"))
			ctx, cancel := context.WithCancel(context.Background())

			errCh := make(chan error, 1)
			go func() {
				errCh <- srv.Run(ctx)
			}()

			time.Sleep(50 * time.Millisecond)
			cancel()

			Eventually(errCh, "3s").Should(Receive(BeNil()))
		})

		It("serves /api/v1/healthz while running", func() {
			srv := server.New(validParsed("127.0.0.1:19090"), handler.Mount)
			ctx, cancel := context.WithCancel(context.Background())
			DeferCleanup(cancel)

			addrCh := make(chan string, 1)
			go func() { _ = srv.Run(ctx, func(addr string) { addrCh <- addr }) }()

			Eventually(func() error {
				resp, err := http.Get("http://127.0.0.1:19090/api/v1/healthz") //nolint:noctx
				if err != nil {
					return err
				}
				_ = resp.Body.Close()
				return nil
			}, "3s", "50ms").Should(Succeed())

			resp, err := http.Get("http://127.0.0.1:19090/api/v1/healthz") //nolint:noctx
			Expect(err).NotTo(HaveOccurred())
			defer func() { _ = resp.Body.Close() }()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})
	})
})
