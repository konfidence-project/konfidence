package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/sync/singleflight"
)

const maximumDiscoveryDocumentSize = 1024 * 1024

type verifierKey struct {
	endpoint string
	audience string
}

type verifiedToken struct {
	subject string
	claims  map[string]any
}

type tokenVerifier interface {
	Verify(ctx context.Context, rawToken string, endpoint string, audience string) (*verifiedToken, error)
}

type oidcTokenVerifier struct {
	httpClient *http.Client
	mu         sync.RWMutex
	verifiers  map[verifierKey]*oidc.IDTokenVerifier
	requests   singleflight.Group
}

type discoveryDocument struct {
	Issuer  string `json:"issuer"`
	JWKSURI string `json:"jwks_uri"`
}

func newOIDCTokenVerifier() tokenVerifier {
	return &oidcTokenVerifier{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(request *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return errors.New("too many redirects")
				}
				if request.URL.Scheme != "https" {
					return errors.New("OIDC redirects must use HTTPS")
				}
				return nil
			},
		},
		verifiers: make(map[verifierKey]*oidc.IDTokenVerifier),
	}
}

func (v *oidcTokenVerifier) Verify(ctx context.Context, rawToken string, endpoint string, audience string) (*verifiedToken, error) {
	key := verifierKey{endpoint: endpoint, audience: audience}
	verifier, err := v.getVerifier(ctx, key)
	if err != nil {
		return nil, err
	}

	idToken, err := verifier.Verify(ctx, rawToken)
	if err != nil {
		return nil, ErrInvalidBearerToken
	}

	var claims map[string]any
	if err := idToken.Claims(&claims); err != nil {
		return nil, ErrInvalidBearerToken
	}

	if idToken.Subject == "" {
		return nil, ErrInvalidBearerToken
	}
	return &verifiedToken{subject: idToken.Subject, claims: claims}, nil
}

func (v *oidcTokenVerifier) getVerifier(ctx context.Context, key verifierKey) (*oidc.IDTokenVerifier, error) {
	v.mu.RLock()
	verifier := v.verifiers[key]
	v.mu.RUnlock()
	if verifier != nil {
		return verifier, nil
	}

	result, err, _ := v.requests.Do(
		key.endpoint+"\x00"+key.audience,
		func() (any, error) {
			v.mu.RLock()
			cached := v.verifiers[key]
			v.mu.RUnlock()

			if cached != nil {
				return cached, nil
			}

			created, err := v.createVerifier(ctx, key)
			if err != nil {
				return nil, err
			}

			v.mu.Lock()
			v.verifiers[key] = created
			v.mu.Unlock()

			return created, nil
		},
	)

	if err != nil {
		return nil, err
	}
	return result.(*oidc.IDTokenVerifier), nil
}

func (v *oidcTokenVerifier) createVerifier(ctx context.Context, key verifierKey) (*oidc.IDTokenVerifier, error) {
	if err := validateHTTPSURL(key.endpoint); err != nil {
		return nil, fmt.Errorf("invalid OIDC discovery endpoint: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, key.endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating discovery request failed: %w", err)
	}

	response, err := v.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("retrieving OIDC discovery document failed: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OIDC discovery returned HTTP %d", response.StatusCode)
	}

	documentBytes, err := io.ReadAll(io.LimitReader(response.Body, maximumDiscoveryDocumentSize+1))
	if err != nil {
		return nil, fmt.Errorf("reading OIDC discovery document failed: %w", err)
	}
	if len(documentBytes) > maximumDiscoveryDocumentSize {
		return nil, errors.New("OIDC discovery document is too large")
	}

	var document discoveryDocument
	if err := json.Unmarshal(documentBytes, &document); err != nil {
		return nil, fmt.Errorf("decoding OIDC discovery document: %w", err)
	}
	if document.Issuer == "" {
		return nil, errors.New("OIDC discovery document has no issuer")
	}
	if err := validateHTTPSURL(document.Issuer); err != nil {
		return nil, fmt.Errorf("invalid OIDC issuer: %w", err)
	}
	if err := validateHTTPSURL(document.JWKSURI); err != nil {
		return nil, fmt.Errorf("invalid OIDC JWKS URI: %w", err)
	}

	keySetContext := oidc.ClientContext(context.Background(), v.httpClient)
	keySet := oidc.NewRemoteKeySet(keySetContext, document.JWKSURI)

	return oidc.NewVerifier(document.Issuer, keySet, &oidc.Config{ClientID: key.audience}), nil
}

func validateHTTPSURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if parsed.Scheme != "https" || parsed.Host == "" {
		return errors.New("URL must use HTTPS and include a host")
	}

	return nil
}
