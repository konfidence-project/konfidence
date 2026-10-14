package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-jose/go-jose/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	discoveryPath = "/discovery"
	jwksPath      = "/jwks"
)

func newTestOIDCTokenVerifier(client *http.Client, ttl time.Duration) *oidcTokenVerifier {
	GinkgoHelper()
	verifier, ok := newOIDCTokenVerifier(ttl).(*oidcTokenVerifier)
	Expect(ok).To(BeTrue())
	verifier.httpClient = client
	return verifier
}

func newTestOIDCProvider(publicKey *rsa.PublicKey, keyID string) (*httptest.Server, *atomic.Int32, *atomic.Int32) {
	GinkgoHelper()
	discoveryCalls := &atomic.Int32{}
	jwksCalls := &atomic.Int32{}

	var server *httptest.Server
	server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer GinkgoRecover()
		switch r.URL.Path {
		case discoveryPath:
			discoveryCalls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			Expect(json.NewEncoder(w).Encode(discoveryDocument{
				Issuer:  server.URL,
				JWKSURI: server.URL + jwksPath,
			})).To(Succeed())
		case jwksPath:
			jwksCalls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			Expect(json.NewEncoder(w).Encode(jose.JSONWebKeySet{
				Keys: []jose.JSONWebKey{{
					Key:       publicKey,
					KeyID:     keyID,
					Algorithm: string(jose.RS256),
					Use:       "sig",
				}},
			})).To(Succeed())
		default:
			http.NotFound(w, r)
		}
	}))
	DeferCleanup(server.Close)
	return server, discoveryCalls, jwksCalls
}

func validWorkloadClaims(issuer, audience string) map[string]any {
	return map[string]any{
		"iss": issuer,
		"aud": audience,
		"sub": "workload-subject",
		"iat": time.Now().Add(-time.Minute).Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
	}
}

func signWorkloadToken(privateKey *rsa.PrivateKey, keyID string, claims map[string]any) string {
	GinkgoHelper()
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: privateKey},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader(jose.HeaderKey("kid"), keyID),
	)
	Expect(err).NotTo(HaveOccurred())
	payload, err := json.Marshal(claims)
	Expect(err).NotTo(HaveOccurred())
	signed, err := signer.Sign(payload)
	Expect(err).NotTo(HaveOccurred())
	rawToken, err := signed.CompactSerialize()
	Expect(err).NotTo(HaveOccurred())
	return rawToken
}

func generateKey() *rsa.PrivateKey {
	GinkgoHelper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	Expect(err).NotTo(HaveOccurred())
	return privateKey
}

var _ = Describe("OIDC token verifier", func() {
	It("verifies tokens and shares provider state across audiences", func() {
		privateKey := generateKey()
		const keyID = "test-key"

		var discoveryCalls atomic.Int32
		var jwksCalls atomic.Int32
		var server *httptest.Server
		server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer GinkgoRecover()
			switch r.URL.Path {
			case "/.well-known/openid-configuration":
				discoveryCalls.Add(1)
				w.Header().Set("Content-Type", "application/json")
				Expect(json.NewEncoder(w).Encode(discoveryDocument{
					Issuer: server.URL, JWKSURI: server.URL + jwksPath,
				})).To(Succeed())
			case jwksPath:
				jwksCalls.Add(1)
				w.Header().Set("Content-Type", "application/json")
				Expect(json.NewEncoder(w).Encode(jose.JSONWebKeySet{
					Keys: []jose.JSONWebKey{{
						Key: &privateKey.PublicKey, KeyID: keyID,
						Algorithm: string(jose.RS256), Use: "sig",
					}},
				})).To(Succeed())
			default:
				http.NotFound(w, r)
			}
		}))
		DeferCleanup(server.Close)
		verifier := newTestOIDCTokenVerifier(server.Client(), time.Hour)

		for _, test := range []struct {
			subject  string
			clientID string
			audience string
		}{
			{
				subject:  "workload-subject",
				clientID: "workload-client",
				audience: "workload-audience",
			},
			{
				subject:  "second-workload-subject",
				audience: "second-workload-audience",
			},
		} {
			claims := validWorkloadClaims(server.URL, test.audience)
			claims["sub"] = test.subject
			claims["repository"] = "konfidence-project/konfidence"
			if test.clientID != "" {
				claims["client_id"] = test.clientID
			}
			token, err := verifier.Verify(
				context.Background(),
				signWorkloadToken(privateKey, keyID, claims),
				server.URL+"/.well-known/openid-configuration",
				[]string{test.audience},
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(token.subject).To(Equal(test.subject))
			Expect(token.claims).To(HaveKeyWithValue("repository", "konfidence-project/konfidence"))
		}
		Expect(discoveryCalls.Load()).To(Equal(int32(1)))
		Expect(jwksCalls.Load()).To(Equal(int32(1)))
	})

	Describe("invalid claims", func() {
		var privateKey *rsa.PrivateKey
		var server *httptest.Server
		const keyID = "test-key"

		BeforeEach(func() {
			privateKey = generateKey()
			server, _, _ = newTestOIDCProvider(&privateKey.PublicKey, keyID)
		})

		DescribeTable("rejects the token",
			func(claims func(string) map[string]any) {
				verifier := newTestOIDCTokenVerifier(server.Client(), time.Hour)
				token, err := verifier.Verify(
					context.Background(),
					signWorkloadToken(privateKey, keyID, claims(server.URL)),
					server.URL+discoveryPath,
					[]string{"konfidence-api"},
				)
				Expect(token).To(BeNil())
				Expect(err).To(MatchError(ErrInvalidBearerToken))
			},
			Entry("with the wrong audience", func(serverURL string) map[string]any {
				return map[string]any{"iss": serverURL, "aud": "different-audience", "sub": "workload-subject", "exp": time.Now().Add(time.Hour).Unix()}
			}),
			Entry("with the wrong issuer", func(string) map[string]any {
				return map[string]any{"iss": "https://different.example", "aud": "konfidence-api", "sub": "workload-subject", "exp": time.Now().Add(time.Hour).Unix()}
			}),
			Entry("when expired", func(serverURL string) map[string]any {
				return map[string]any{"iss": serverURL, "aud": "konfidence-api", "sub": "workload-subject", "exp": time.Now().Add(-time.Hour).Unix()}
			}),
			Entry("when the subject is missing", func(serverURL string) map[string]any {
				return map[string]any{"iss": serverURL, "aud": "konfidence-api", "client_id": "workload-client", "exp": time.Now().Add(time.Hour).Unix()}
			}),
			Entry("when the subject is empty", func(serverURL string) map[string]any {
				return map[string]any{"iss": serverURL, "aud": "konfidence-api", "sub": "", "client_id": "workload-client", "exp": time.Now().Add(time.Hour).Unix()}
			}),
		)
	})

	DescribeTable("validates HTTPS URLs",
		func(rawURL string, valid bool) {
			err := validateHTTPSURL(rawURL)
			if valid {
				Expect(err).NotTo(HaveOccurred())
			} else {
				Expect(err).To(HaveOccurred())
			}
		},
		Entry("an HTTPS URL", "https://issuer.example/.well-known/openid-configuration", true),
		Entry("an HTTPS URL with a port", "https://issuer.example:8443/discovery", true),
		Entry("rejects HTTP", "http://issuer.example/discovery", false),
		Entry("rejects a relative URL", "/.well-known/openid-configuration", false),
		Entry("rejects a missing host", "https:///discovery", false),
		Entry("rejects an empty URL", "", false),
		Entry("rejects a malformed URL", "https://issuer.example/%", false),
	)

	It("rejects an oversized discovery document", func() {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			defer GinkgoRecover()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(strings.Repeat("x", maximumDiscoveryDocumentSize+1)))
			Expect(err).NotTo(HaveOccurred())
		}))
		DeferCleanup(server.Close)
		result, err := newTestOIDCTokenVerifier(server.Client(), time.Hour).createProvider(context.Background(), server.URL+discoveryPath)
		Expect(result).To(BeNil())
		Expect(err).To(MatchError("OIDC discovery document is too large"))
	})

	It("rejects a discovery document without an issuer", func() {
		var server *httptest.Server
		server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer GinkgoRecover()
			switch r.URL.Path {
			case discoveryPath:
				w.Header().Set("Content-Type", "application/json")
				Expect(json.NewEncoder(w).Encode(discoveryDocument{JWKSURI: server.URL + jwksPath})).To(Succeed())
			case jwksPath:
				http.Error(w, "JWKS must not be requested", http.StatusInternalServerError)
			default:
				http.NotFound(w, r)
			}
		}))
		DeferCleanup(server.Close)
		result, err := newTestOIDCTokenVerifier(server.Client(), time.Hour).createProvider(context.Background(), server.URL+discoveryPath)
		Expect(result).To(BeNil())
		Expect(err).To(MatchError("OIDC discovery document has no issuer"))
	})

	It("rejects an insecure JWKS URL", func() {
		var server *httptest.Server
		server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer GinkgoRecover()
			if r.URL.Path != discoveryPath {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			Expect(json.NewEncoder(w).Encode(discoveryDocument{
				Issuer: server.URL, JWKSURI: "http://issuer.example/jwks",
			})).To(Succeed())
		}))
		DeferCleanup(server.Close)
		result, err := newTestOIDCTokenVerifier(server.Client(), time.Hour).createProvider(context.Background(), server.URL+discoveryPath)
		Expect(result).To(BeNil())
		Expect(err).To(MatchError("invalid OIDC JWKS URI: URL must use HTTPS and include a host"))
	})

	It("expires cached provider state", func() {
		privateKey := generateKey()
		const keyID, audience = "test-key", "konfidence-api"
		server, discoveryCalls, jwksCalls := newTestOIDCProvider(&privateKey.PublicKey, keyID)
		verifier := newTestOIDCTokenVerifier(server.Client(), time.Nanosecond)
		rawToken := signWorkloadToken(privateKey, keyID, validWorkloadClaims(server.URL, audience))

		for range 2 {
			token, err := verifier.Verify(context.Background(), rawToken, server.URL+discoveryPath, []string{audience})
			Expect(err).NotTo(HaveOccurred())
			Expect(token).NotTo(BeNil())
		}
		Expect(discoveryCalls.Load()).To(Equal(int32(2)))
		Expect(jwksCalls.Load()).To(Equal(int32(2)))
	})

	It("keeps cached provider state after a signature failure", func() {
		trustedKey := generateKey()
		untrustedKey := generateKey()
		const (
			trustedKeyID   = "trusted-key"
			untrustedKeyID = "untrusted-key"
			audience       = "konfidence-api"
		)
		server, discoveryCalls, jwksCalls := newTestOIDCProvider(&trustedKey.PublicKey, trustedKeyID)
		verifier := newTestOIDCTokenVerifier(server.Client(), time.Hour)
		claims := validWorkloadClaims(server.URL, audience)
		validToken := signWorkloadToken(trustedKey, trustedKeyID, claims)
		_, err := verifier.Verify(context.Background(), validToken, server.URL+discoveryPath, []string{audience})
		Expect(err).NotTo(HaveOccurred())

		invalidToken := signWorkloadToken(untrustedKey, untrustedKeyID, claims)
		_, err = verifier.Verify(context.Background(), invalidToken, server.URL+discoveryPath, []string{audience})
		Expect(err).To(MatchError(ErrInvalidBearerToken))

		// The remote key set refreshes after the signature failure; the provider state remains cached.
		for range 2 {
			_, err = verifier.Verify(context.Background(), validToken, server.URL+discoveryPath, []string{audience})
			Expect(err).NotTo(HaveOccurred())
		}
		Expect(discoveryCalls.Load()).To(Equal(int32(1)))
		Expect(jwksCalls.Load()).To(Equal(int32(2)))
	})

	It("keeps the cache after a claim failure", func() {
		privateKey := generateKey()
		const keyID, audience = "test-key", "konfidence-api"
		server, discoveryCalls, jwksCalls := newTestOIDCProvider(&privateKey.PublicKey, keyID)
		verifier := newTestOIDCTokenVerifier(server.Client(), time.Hour)
		validToken := signWorkloadToken(privateKey, keyID, validWorkloadClaims(server.URL, audience))
		_, err := verifier.Verify(context.Background(), validToken, server.URL+discoveryPath, []string{audience})
		Expect(err).NotTo(HaveOccurred())

		invalidToken := signWorkloadToken(privateKey, keyID, validWorkloadClaims(server.URL, "different-audience"))
		_, err = verifier.Verify(context.Background(), invalidToken, server.URL+discoveryPath, []string{audience})
		Expect(err).To(MatchError(ErrInvalidBearerToken))
		_, err = verifier.Verify(context.Background(), validToken, server.URL+discoveryPath, []string{audience})
		Expect(err).NotTo(HaveOccurred())
		Expect(discoveryCalls.Load()).To(Equal(int32(1)))
		Expect(jwksCalls.Load()).To(Equal(int32(1)))
	})

	It("shares provider state across audiences for the same endpoint", func() {
		privateKey := generateKey()
		const keyID = "test-key"

		server, discoveryCalls, jwksCalls := newTestOIDCProvider(
			&privateKey.PublicKey,
			keyID,
		)
		verifier := newTestOIDCTokenVerifier(server.Client(), time.Hour)

		for _, audience := range []string{"audience-a", "audience-b"} {
			rawToken := signWorkloadToken(
				privateKey,
				keyID,
				validWorkloadClaims(server.URL, audience),
			)

			token, err := verifier.Verify(context.Background(), rawToken, server.URL+discoveryPath, []string{audience})

			Expect(err).NotTo(HaveOccurred())
			Expect(token).NotTo(BeNil())
		}

		Expect(discoveryCalls.Load()).To(Equal(int32(1)))
		Expect(jwksCalls.Load()).To(Equal(int32(1)))
	})

	It("returns every configured audience present in the verified token", func() {
		privateKey := generateKey()
		const keyID = "test-key"

		server, discoveryCalls, jwksCalls := newTestOIDCProvider(&privateKey.PublicKey, keyID)
		verifier := newTestOIDCTokenVerifier(server.Client(), time.Hour)
		claims := validWorkloadClaims(server.URL, "unused")
		claims["aud"] = []string{"audience-a", "audience-b", "unconfigured-audience"}
		rawToken := signWorkloadToken(privateKey, keyID, claims)
		token, err := verifier.Verify(
			context.Background(),
			rawToken,
			server.URL+discoveryPath,
			[]string{
				"audience-a",
				"audience-b",
				"missing-audience",
			},
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(token).NotTo(BeNil())
		Expect(token.audiences).To(ConsistOf("audience-a", "audience-b"))
		Expect(discoveryCalls.Load()).To(Equal(int32(1)))
		Expect(jwksCalls.Load()).To(Equal(int32(1)))
	})

	It("rejects a token without a configured verified audience", func() {
		privateKey := generateKey()
		const keyID = "test-key"

		server, discoveryCalls, jwksCalls := newTestOIDCProvider(&privateKey.PublicKey, keyID)
		verifier := newTestOIDCTokenVerifier(server.Client(), time.Hour)

		rawToken := signWorkloadToken(privateKey, keyID, validWorkloadClaims(server.URL, "different-audience"))
		token, err := verifier.Verify(context.Background(), rawToken, server.URL+discoveryPath, []string{"audience-a", "audience-b"})

		Expect(token).To(BeNil())
		Expect(err).To(MatchError(ErrInvalidBearerToken))

		// Signature verification happens before the manual audience check.
		Expect(discoveryCalls.Load()).To(Equal(int32(1)))
		Expect(jwksCalls.Load()).To(Equal(int32(1)))
	})
})
