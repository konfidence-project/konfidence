package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/maypok86/otter/v2"
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
	verifiers  *otter.Cache[verifierKey, *oidc.IDTokenVerifier]
}

type discoveryDocument struct {
	Issuer  string `json:"issuer"`
	JWKSURI string `json:"jwks_uri"`
}

func newOIDCTokenVerifier(jwksCacheTTL time.Duration) tokenVerifier {
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
		verifiers: otter.Must(&otter.Options[verifierKey, *oidc.IDTokenVerifier]{
			ExpiryCalculator: otter.ExpiryCreating[verifierKey, *oidc.IDTokenVerifier](jwksCacheTTL),
		}),
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
	return v.verifiers.Get(ctx, key, otter.LoaderFunc[verifierKey, *oidc.IDTokenVerifier](v.createVerifier))
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
	remoteKeySet := oidc.NewRemoteKeySet(keySetContext, document.JWKSURI)
	return oidc.NewVerifier(
		document.Issuer,
		remoteKeySet,
		&oidc.Config{ClientID: key.audience},
	), nil
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
