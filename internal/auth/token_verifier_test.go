package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-jose/go-jose/v4"
)

func signWorkloadToken(
	t *testing.T,
	privateKey *rsa.PrivateKey,
	keyID string,
	claims map[string]any,
) string {
	t.Helper()

	signer, err := jose.NewSigner(
		jose.SigningKey{
			Algorithm: jose.RS256,
			Key:       privateKey,
		},
		(&jose.SignerOptions{}).
			WithType("JWT").
			WithHeader(jose.HeaderKey("kid"), keyID),
	)
	if err != nil {
		t.Fatal(err)
	}

	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}

	signed, err := signer.Sign(payload)
	if err != nil {
		t.Fatal(err)
	}

	rawToken, err := signed.CompactSerialize()
	if err != nil {
		t.Fatal(err)
	}

	return rawToken
}

func TestOIDCTokenVerifierVerifiesAndCachesProvider(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	const (
		keyID    = "test-key"
		audience = "konfidence-api"
	)

	var discoveryCalls atomic.Int32
	var jwksCalls atomic.Int32
	var server *httptest.Server

	server = httptest.NewTLSServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		const jwksPath = "/jwks"
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			discoveryCalls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(discoveryDocument{
				Issuer:  server.URL,
				JWKSURI: server.URL + jwksPath,
			}); err != nil {
				t.Errorf("encoding discovery document: %v", err)
			}

		case jwksPath:
			jwksCalls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(
				jose.JSONWebKeySet{
					Keys: []jose.JSONWebKey{{
						Key:       &privateKey.PublicKey,
						KeyID:     keyID,
						Algorithm: string(jose.RS256),
						Use:       "sig",
					}},
				},
			); err != nil {
				t.Errorf("encoding JWKS: %v", err)
			}

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	verifier := &oidcTokenVerifier{
		httpClient: server.Client(),
		verifiers:  make(map[verifierKey]*oidc.IDTokenVerifier),
	}

	tests := []struct {
		name        string
		claims      map[string]any
		wantSubject string
	}{
		{
			name: "uses subject claim",
			claims: map[string]any{
				"iss":        server.URL,
				"aud":        audience,
				"sub":        "workload-subject",
				"client_id":  "workload-client",
				"repository": "konfidence-project/konfidence",
				"iat":        time.Now().Add(-time.Minute).Unix(),
				"exp":        time.Now().Add(time.Hour).Unix(),
			},
			wantSubject: "workload-subject",
		},
		{
			name: "uses subject with cached provider",
			claims: map[string]any{
				"iss":        server.URL,
				"aud":        audience,
				"sub":        "second-workload-subject",
				"repository": "konfidence-project/konfidence",
				"iat":        time.Now().Add(-time.Minute).Unix(),
				"exp":        time.Now().Add(time.Hour).Unix(),
			},
			wantSubject: "second-workload-subject",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rawToken := signWorkloadToken(
				t,
				privateKey,
				keyID,
				test.claims,
			)

			token, err := verifier.Verify(
				context.Background(),
				rawToken,
				server.URL+"/.well-known/openid-configuration",
				audience,
			)
			if err != nil {
				t.Fatal(err)
			}
			if token.subject != test.wantSubject {
				t.Fatalf(
					"expected subject %q, got %q",
					test.wantSubject,
					token.subject,
				)
			}
			if token.claims["repository"] !=
				"konfidence-project/konfidence" {
				t.Fatalf(
					"unexpected claims: %v",
					token.claims,
				)
			}
		})
	}

	if got := discoveryCalls.Load(); got != 1 {
		t.Fatalf(
			"expected one discovery request, got %d",
			got,
		)
	}
	if got := jwksCalls.Load(); got != 1 {
		t.Fatalf(
			"expected one JWKS request, got %d",
			got,
		)
	}
}

func TestOIDCTokenVerifierRejectsInvalidClaims(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	const keyID = "test-key"

	var server *httptest.Server
	const discoveryPath = "/discovery"
	server = httptest.NewTLSServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		switch r.URL.Path {
		case discoveryPath:
			_ = json.NewEncoder(w).Encode(discoveryDocument{
				Issuer:  server.URL,
				JWKSURI: server.URL + "/jwks",
			})

		case "/jwks":
			_ = json.NewEncoder(w).Encode(
				jose.JSONWebKeySet{
					Keys: []jose.JSONWebKey{{
						Key:       &privateKey.PublicKey,
						KeyID:     keyID,
						Algorithm: string(jose.RS256),
						Use:       "sig",
					}},
				},
			)

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	tests := map[string]map[string]any{
		"wrong audience": {
			"iss": server.URL,
			"aud": "different-audience",
			"sub": "workload-subject",
			"exp": time.Now().Add(time.Hour).Unix(),
		},
		"wrong issuer": {
			"iss": "https://different.example",
			"aud": "konfidence-api",
			"sub": "workload-subject",
			"exp": time.Now().Add(time.Hour).Unix(),
		},
		"expired": {
			"iss": server.URL,
			"aud": "konfidence-api",
			"sub": "workload-subject",
			"exp": time.Now().Add(-time.Hour).Unix(),
		},
		"missing subject with client ID": {
			"iss":       server.URL,
			"aud":       "konfidence-api",
			"client_id": "workload-client",
			"exp":       time.Now().Add(time.Hour).Unix(),
		},
		"empty subject with client ID": {
			"iss":       server.URL,
			"aud":       "konfidence-api",
			"sub":       "",
			"client_id": "workload-client",
			"exp":       time.Now().Add(time.Hour).Unix(),
		},
	}

	for name, claims := range tests {
		t.Run(name, func(t *testing.T) {
			verifier := &oidcTokenVerifier{
				httpClient: server.Client(),
				verifiers: make(
					map[verifierKey]*oidc.IDTokenVerifier,
				),
			}
			rawToken := signWorkloadToken(
				t,
				privateKey,
				keyID,
				claims,
			)

			token, err := verifier.Verify(
				context.Background(),
				rawToken,
				server.URL+discoveryPath,
				"konfidence-api",
			)

			if token != nil {
				t.Fatalf(
					"expected nil token, got %+v",
					token,
				)
			}
			if !errors.Is(err, ErrInvalidBearerToken) {
				t.Fatalf(
					"expected ErrInvalidBearerToken, got %v",
					err,
				)
			}
		})
	}
}

func TestValidateHTTPSURL(t *testing.T) {
	tests := map[string]struct {
		rawURL  string
		wantErr bool
	}{
		"valid HTTPS URL": {
			rawURL: "https://issuer.example/.well-known/openid-configuration",
		},
		"valid HTTPS URL with port": {
			rawURL: "https://issuer.example:8443/discovery",
		},
		"rejects HTTP": {
			rawURL:  "http://issuer.example/discovery",
			wantErr: true,
		},
		"rejects relative URL": {
			rawURL:  "/.well-known/openid-configuration",
			wantErr: true,
		},
		"rejects missing host": {
			rawURL:  "https:///discovery",
			wantErr: true,
		},
		"rejects empty URL": {
			rawURL:  "",
			wantErr: true,
		},
		"rejects malformed URL": {
			rawURL:  "https://issuer.example/%",
			wantErr: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateHTTPSURL(test.rawURL)

			if test.wantErr && err == nil {
				t.Fatalf(
					"validateHTTPSURL(%q) succeeded, expected error",
					test.rawURL,
				)
			}
			if !test.wantErr && err != nil {
				t.Fatalf(
					"validateHTTPSURL(%q) returned error: %v",
					test.rawURL,
					err,
				)
			}
		})
	}
}

func TestOIDCTokenVerifierRejectsOversizedDiscoveryDocument(
	t *testing.T,
) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		_ *http.Request,
	) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, err := w.Write([]byte(strings.Repeat(
			"x",
			maximumDiscoveryDocumentSize+1,
		)))
		if err != nil {
			t.Errorf("writing discovery response: %v", err)
		}
	}))
	defer server.Close()

	verifier := &oidcTokenVerifier{
		httpClient: server.Client(),
		verifiers:  make(map[verifierKey]*oidc.IDTokenVerifier),
	}

	result, err := verifier.createVerifier(
		context.Background(),
		verifierKey{
			endpoint: server.URL + "/discovery",
			audience: "konfidence-api",
		},
	)

	if result != nil {
		t.Fatalf("expected nil verifier, got %v", result)
	}
	if err == nil {
		t.Fatal("expected oversized discovery document to be rejected")
	}
	if err.Error() != "OIDC discovery document is too large" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOIDCTokenVerifierRejectsDiscoveryWithoutIssuer(
	t *testing.T,
) {
	var server *httptest.Server

	server = httptest.NewTLSServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		switch r.URL.Path {
		case "/discovery":
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(discoveryDocument{
				JWKSURI: server.URL + "/jwks",
			}); err != nil {
				t.Errorf("encoding discovery document: %v", err)
			}

		case "/jwks":
			http.Error(
				w,
				"JWKS must not be requested",
				http.StatusInternalServerError,
			)

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	verifier := &oidcTokenVerifier{
		httpClient: server.Client(),
		verifiers:  make(map[verifierKey]*oidc.IDTokenVerifier),
	}

	result, err := verifier.createVerifier(
		context.Background(),
		verifierKey{
			endpoint: server.URL + "/discovery",
			audience: "konfidence-api",
		},
	)

	if result != nil {
		t.Fatalf("expected nil verifier, got %v", result)
	}
	if err == nil {
		t.Fatal("expected discovery document without issuer to be rejected")
	}
	if err.Error() != "OIDC discovery document has no issuer" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOIDCTokenVerifierRejectsInsecureJWKSURL(
	t *testing.T,
) {
	var server *httptest.Server

	server = httptest.NewTLSServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		if r.URL.Path != "/discovery" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(discoveryDocument{
			Issuer:  server.URL,
			JWKSURI: "http://issuer.example/jwks",
		}); err != nil {
			t.Errorf("encoding discovery document: %v", err)
		}
	}))
	defer server.Close()

	verifier := &oidcTokenVerifier{
		httpClient: server.Client(),
		verifiers:  make(map[verifierKey]*oidc.IDTokenVerifier),
	}

	result, err := verifier.createVerifier(
		context.Background(),
		verifierKey{
			endpoint: server.URL + "/discovery",
			audience: "konfidence-api",
		},
	)

	if result != nil {
		t.Fatalf("expected nil verifier, got %v", result)
	}
	if err == nil {
		t.Fatal("expected insecure JWKS URL to be rejected")
	}

	expected := "invalid OIDC JWKS URI: " +
		"URL must use HTTPS and include a host"
	if err.Error() != expected {
		t.Fatalf(
			"expected error %q, got %q",
			expected,
			err.Error(),
		)
	}
}
