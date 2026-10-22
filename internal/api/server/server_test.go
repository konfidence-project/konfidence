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

func validParsed() config.Parsed {
	cfg := config.Config{
		Server: config.ServerConfig{
			Addr: "127.0.0.1:0", LogLevel: "error", ReadTimeout: "5s", WriteTimeout: "5s", ShutdownTimeout: "2s",
		},
		OIDC: config.OIDCConfig{
			Enabled:   true,
			IssuerURL: "http://localhost:5556/oidc", ClientID: "konfidence", ClientSecret: "a secret",
			RedirectURL: "http://localhost:8090/api/v1/auth/callback", PKCEEnabled: true, StateExpiration: "15m",
			AllowedReturnHosts: []string{"localhost"},
			JWKSCacheTTL:       "15m",
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
			parsed := validParsed()
			Expect(parsed.Server.ShutdownTimeout).To(Equal(2 * time.Second))
			Expect(parsed.OIDC.StateExpiration).To(Equal(15 * time.Minute))
			Expect(parsed.Session.Expiration).To(Equal(12 * time.Hour))
		})

		It("starts and stops cleanly when context is cancelled", func() {
			srv := server.New(validParsed(), getLogger(), http.NotFoundHandler())
			ctx, cancel := context.WithCancel(context.Background())

			addrCh := make(chan string, 1)
			errCh := make(chan error, 1)
			go func() {
				errCh <- srv.ListenAndServe(ctx, func(addr string) {
					addrCh <- addr
				})
			}()

			var addr string
			Eventually(addrCh, 3*time.Second).Should(Receive(&addr))

			cancel()
			Eventually(errCh, 3*time.Second).Should(Receive(BeNil()))
		})

		It("serves /healthz while running", func() {
			srv := server.New(validParsed(), getLogger(), http.NotFoundHandler())
			ctx, cancel := context.WithCancel(context.Background())

			addrCh := make(chan string, 1)
			errCh := make(chan error, 1)
			go func() {
				errCh <- srv.ListenAndServe(ctx, func(addr string) {
					addrCh <- addr
				})
			}()

			var addr string
			Eventually(addrCh, 3*time.Second).Should(Receive(&addr))

			resp, err := http.Get("http://" + addr + "/healthz") //nolint:noctx
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(resp.Body.Close()).To(Succeed())

			cancel()
			Eventually(errCh, 3*time.Second).Should(Receive(BeNil()))

		})

		It("serves the dashboard and preserves API 404 responses", func() {
			uiDir := GinkgoT().TempDir()
			Expect(os.WriteFile(filepath.Join(uiDir, "index.html"), []byte("Konfidence SPA"), 0o600)).To(Succeed())
			cfg := validParsed()
			cfg.Server.UIAssetPath = uiDir

			srv := server.New(cfg, getLogger(), http.NotFoundHandler())
			ctx, cancel := context.WithCancel(context.Background())

			addrCh := make(chan string, 1)
			errCh := make(chan error, 1)
			go func() {
				errCh <- srv.ListenAndServe(ctx, func(addr string) {
					addrCh <- addr
				})
			}()

			var addr string
			Eventually(addrCh, 3*time.Second).Should(Receive(&addr))

			resp, err := http.Get("http://" + addr + "/projects/example/landscape") //nolint:noctx
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(resp.Body.Close()).To(Succeed())

			resp, err = http.Get("http://" + addr + "/api/v1/unknown") //nolint:noctx
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			Expect(resp.Header.Get("Content-Type")).NotTo(ContainSubstring("text/html"))
			Expect(resp.Body.Close()).To(Succeed())

			cancel()
			Eventually(errCh, 3*time.Second).Should(Receive(BeNil()))
		})

		It("serves /healthz over TLS when tls-enabled is true", func() {
			certPEM, keyPEM := generateTestCert()
			dir := GinkgoT().TempDir()
			certFile := filepath.Join(dir, "tls.crt")
			keyFile := filepath.Join(dir, "tls.key")

			Expect(os.WriteFile(certFile, certPEM, 0o600)).To(Succeed())
			Expect(os.WriteFile(keyFile, keyPEM, 0o600)).To(Succeed())

			cfg := validParsed()
			cfg.Server.TLSCertFile = certFile
			cfg.Server.TLSKeyFile = keyFile

			srv := server.New(cfg, getLogger(), http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			ctx, cancel := context.WithCancel(context.Background())
			DeferCleanup(cancel)

			addrCh := make(chan string, 1)
			errCh := make(chan error, 1)

			go func() {
				errCh <- srv.ListenAndServe(ctx, func(addr string) {
					addrCh <- addr
				})
			}()

			var addr string
			Eventually(addrCh, 3*time.Second).Should(Receive(&addr))
			Expect(addr).NotTo(BeEmpty())

			tlsClient := &http.Client{Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}}
			DeferCleanup(tlsClient.CloseIdleConnections)

			resp, err := tlsClient.Get("https://" + addr + "/healthz")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(resp.Body.Close()).To(Succeed())

			cancel()
			Eventually(errCh, 3*time.Second).Should(Receive(BeNil()))
		})
	})
})
