package server_test

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/konfidence-project/konfidence/internal/api/server"
)

func validParsed(addr string) config.Parsed {
	cfg := config.Config{
		Addr:                  addr,
		LogLevel:              "error",
		ReadTimeout:           "5s",
		WriteTimeout:          "5s",
		ShutdownTimeout:       "2s",
		OIDCIssuerURL:         "http://localhost:5556/oidc",
		OIDCClientID:          "konfidence",
		OIDCClientSecret:      "a secret",
		OIDCRedirectURL:       "http://localhost:8090/api/v1/auth/callback",
		OIDCPKCEEnabled:       true,
		OIDCStateExpiration:   "15m",
		SessionCookieName:     "kden-session",
		SessionCookieHTTPOnly: true,
		SessionCookieSecure:   false,
		SessionCookieSameSite: "SameSiteStrictMode",
		SessionExpiry:         "12h",
	}
	parsed, err := cfg.Validate()
	if err != nil {
		panic(err)
	}
	return parsed
}

func getLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

var _ = Describe("Server", func() {
	Describe("Run", func() {
		It("uses the parsed shutdown timeout from config", func() {
			parsed := validParsed("127.0.0.1:0")
			Expect(parsed.ShutdownTimeout).To(Equal(2 * time.Second))
			Expect(parsed.OIDCStateExpiration).To(Equal(15 * time.Minute))
			Expect(parsed.SessionExpiry).To(Equal(12 * time.Hour))
		})

		It("starts and stops cleanly when context is cancelled", func() {
			srv := server.New(validParsed("127.0.0.1:0"), getLogger(), http.NotFoundHandler())
			ctx, cancel := context.WithCancel(context.Background())

			errCh := make(chan error, 1)
			go func() {
				errCh <- srv.ListenAndServe(ctx)
			}()

			time.Sleep(50 * time.Millisecond)
			cancel()

			Eventually(errCh, "3s").Should(Receive(BeNil()))
		})

		It("serves /healthz while running", func() {
			srv := server.New(validParsed("127.0.0.1:19090"), getLogger(), http.NotFoundHandler())
			ctx, cancel := context.WithCancel(context.Background())
			DeferCleanup(cancel)

			addrCh := make(chan string, 1)
			go func() { _ = srv.ListenAndServe(ctx, func(addr string) { addrCh <- addr }) }()

			Eventually(func() error {
				resp, err := http.Get("http://127.0.0.1:19090/healthz") //nolint:noctx
				if err != nil {
					return err
				}
				_ = resp.Body.Close()
				return nil
			}, "3s", "50ms").Should(Succeed())

			resp, err := http.Get("http://127.0.0.1:19090/healthz") //nolint:noctx
			Expect(err).NotTo(HaveOccurred())
			defer func() { _ = resp.Body.Close() }()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})
	})
})
