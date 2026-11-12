package server_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/konfidence-project/konfidence/internal/api/openapi"
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
			DeferCleanup(cancel)

			client := &http.Client{}
			DeferCleanup(client.CloseIdleConnections)

			addrCh := make(chan string, 1)
			errCh := make(chan error, 1)
			go func() {
				errCh <- srv.ListenAndServe(ctx, func(addr string) {
					addrCh <- addr
				})
			}()

			var addr string
			Eventually(addrCh, 3*time.Second).Should(Receive(&addr))

			resp, err := client.Get("http://" + addr + "/projects/example/landscape") //nolint:noctx
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			body, err := io.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Body.Close()).To(Succeed())
			Expect(body).To(Equal([]byte("Konfidence SPA")))

			resp, err = client.Get("http://" + addr + "/api/v1/unknown") //nolint:noctx
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			Expect(resp.Header.Get("Content-Type")).NotTo(ContainSubstring("text/html"))

			_, err = io.Copy(io.Discard, resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Body.Close()).To(Succeed())

			client.CloseIdleConnections()
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

		It("serves the embedded OpenAPI specification and Swagger UI assets", func() {
			srv := server.New(validParsed(), getLogger(), http.NotFoundHandler())
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

			client := &http.Client{
				CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
					return http.ErrUseLastResponse
				},
			}
			DeferCleanup(client.CloseIdleConnections)

			baseURL := "http://" + addr

			redirectResponse, err := client.Get(baseURL + "/docs") //nolint:noctx
			Expect(err).NotTo(HaveOccurred())
			Expect(redirectResponse.StatusCode).To(Equal(http.StatusPermanentRedirect))
			Expect(redirectResponse.Header.Get("Location")).To(Equal("/docs/"))
			Expect(redirectResponse.Body.Close()).To(Succeed())

			specResponse, err := client.Get(baseURL + "/docs/openapi.json") //nolint:noctx
			Expect(err).NotTo(HaveOccurred())
			Expect(specResponse.StatusCode).To(Equal(http.StatusOK))
			Expect(specResponse.Header.Get("Content-Type")).To(Equal("application/json"))

			specBody, err := io.ReadAll(specResponse.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(specResponse.Body.Close()).To(Succeed())

			embeddedSpec, err := openapi.GetSpecJSON()
			Expect(err).NotTo(HaveOccurred())
			Expect(bytes.Equal(specBody, embeddedSpec)).To(BeTrue())

			docsResponse, err := client.Get(baseURL + "/docs/") //nolint:noctx
			Expect(err).NotTo(HaveOccurred())
			Expect(docsResponse.StatusCode).To(Equal(http.StatusOK))
			Expect(docsResponse.Header.Get("Content-Type")).To(ContainSubstring("text/html"))

			docsBody, err := io.ReadAll(docsResponse.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(docsResponse.Body.Close()).To(Succeed())

			docsHTML := string(docsBody)
			Expect(docsHTML).To(ContainSubstring(`"swaggerJsonUrl":"/docs/openapi.json"`))
			Expect(docsHTML).To(ContainSubstring("withCredentials: true"))
			Expect(docsHTML).To(ContainSubstring(`src="/docs/swagger-ui-bundle.js"`))
			Expect(docsHTML).NotTo(ContainSubstring("cdnjs.cloudflare.com"))
			Expect(docsHTML).NotTo(ContainSubstring("unpkg.com"))
			Expect(docsHTML).NotTo(ContainSubstring("cdn.jsdelivr.net"))

			assetResponse, err := client.Get(baseURL + "/docs/swagger-ui.css") //nolint:noctx
			Expect(err).NotTo(HaveOccurred())
			Expect(assetResponse.StatusCode).To(Equal(http.StatusOK))
			Expect(assetResponse.Header.Get("Content-Type")).To(ContainSubstring("text/css"))
			_, err = io.Copy(io.Discard, assetResponse.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(assetResponse.Body.Close()).To(Succeed())

			cancel()
			Eventually(errCh, 3*time.Second).Should(Receive(BeNil()))
		})
	})
})
