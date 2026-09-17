package server_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/konfidence-project/konfidence/internal/api/server"
)

func validParsed(addr string) config.Parsed {
	cfg := config.Config{
		Server: config.ServerConfig{
			Addr: addr, LogLevel: "error", ReadTimeout: "5s", WriteTimeout: "5s", ShutdownTimeout: "2s",
		},
		OIDC: config.OIDCConfig{
			Enabled:   true,
			IssuerURL: "http://localhost:5556/oidc", ClientID: "konfidence", ClientSecret: "a secret",
			RedirectURL: "http://localhost:8090/api/v1/auth/callback", PKCEEnabled: true, StateExpiration: "15m",
			AllowReturnURLs: []string{"http://localhost:3000/auth/callback"},
			JWKSCacheTTL:    "15m",
		},
		Session: config.SessionConfig{
			StorageType:     "in-memory",
			Cookie:          config.SessionCookieConfig{Name: "kden-session", HTTPOnly: true, SameSite: "SameSiteStrictMode"},
			Expiration:      "12h",
			CleanupInterval: "15m",
		},
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

func generateTestCert() (certPEM, keyPEM []byte) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	certDER, _ := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, _ := x509.MarshalECPrivateKey(key)
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return
}

var _ = Describe("Server", func() {
	Describe("Run", func() {
		It("uses the parsed shutdown timeout from config", func() {
			parsed := validParsed("127.0.0.1:0")
			Expect(parsed.Server.ShutdownTimeout).To(Equal(2 * time.Second))
			Expect(parsed.OIDC.StateExpiration).To(Equal(15 * time.Minute))
			Expect(parsed.Session.Expiration).To(Equal(12 * time.Hour))
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

		It("serves the dashboard and preserves API 404 responses", func() {
			uiDir := GinkgoT().TempDir()
			Expect(os.WriteFile(filepath.Join(uiDir, "index.html"), []byte("Konfidence SPA"), 0o600)).To(Succeed())
			cfg := validParsed("127.0.0.1:19091")
			cfg.Server.UIAssetPath = uiDir
			srv := server.New(cfg, getLogger(), http.NotFoundHandler())
			ctx, cancel := context.WithCancel(context.Background())
			DeferCleanup(cancel)

			go func() { _ = srv.ListenAndServe(ctx) }()
			Eventually(func() error {
				resp, err := http.Get("http://127.0.0.1:19091/projects/example/landscape") //nolint:noctx
				if err != nil {
					return err
				}
				_ = resp.Body.Close()
				return nil
			}, "3s", "50ms").Should(Succeed())

			resp, err := http.Get("http://127.0.0.1:19091/projects/example/landscape") //nolint:noctx
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			_ = resp.Body.Close()

			resp, err = http.Get("http://127.0.0.1:19091/api/v1/unknown") //nolint:noctx
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			Expect(resp.Header.Get("Content-Type")).NotTo(ContainSubstring("text/html"))
			_ = resp.Body.Close()
		})

		It("serves /healthz over TLS when tls-enabled is true", func() {
			certPEM, keyPEM := generateTestCert()

			dir := GinkgoT().TempDir()
			certFile := filepath.Join(dir, "tls.crt")
			keyFile := filepath.Join(dir, "tls.key")
			Expect(os.WriteFile(certFile, certPEM, 0600)).To(Succeed())
			Expect(os.WriteFile(keyFile, keyPEM, 0600)).To(Succeed())

			cfg := validParsed("127.0.0.1:0")
			cfg.Server.TLSEnabled = true
			cfg.Server.TLSCertFile = certFile
			cfg.Server.TLSKeyFile = keyFile

			srv := server.New(cfg, getLogger(), http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var addr string
			errCh := make(chan error, 1)
			go func() { errCh <- srv.ListenAndServe(ctx, func(a string) { addr = a }) }()

			tlsClient := &http.Client{Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}}
			Eventually(func() int {
				if addr == "" {
					return 0
				}
				resp, err := tlsClient.Get("https://" + addr + "/healthz")
				if err != nil {
					return 0
				}
				return resp.StatusCode
			}, 3*time.Second, 50*time.Millisecond).Should(Equal(http.StatusOK))

			cancel()
			Eventually(errCh, 3*time.Second).Should(Receive(BeNil()))
		})

	})
})
