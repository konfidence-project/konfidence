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
	"golang.org/x/sync/singleflight"
)

const maximumDiscoveryDocumentSize = 1024 * 1024

type verifiedToken struct {
	subject   string
	claims    map[string]any
	audiences []string
}

type providerState struct {
	issuer string
	keySet oidc.KeySet
}

type discoveryDocument struct {
	Issuer  string `json:"issuer"`
	JWKSURI string `json:"jwks_uri"`
}

type tokenVerifier interface {
	Verify(ctx context.Context, rawToken string, endpoint string, allowedAudiences []string) (*verifiedToken, error)
}
type oidcTokenVerifier struct {
	httpClient     *http.Client
	providers      *otter.Cache[string, *providerState]
	providerLoader singleflight.Group
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
		providers: otter.Must(&otter.Options[string, *providerState]{
			ExpiryCalculator: otter.ExpiryCreating[string, *providerState](jwksCacheTTL),
		}),
	}
}

func (v *oidcTokenVerifier) Verify(ctx context.Context, rawToken string, endpoint string, allowedAudiences []string) (*verifiedToken, error) {
	provider, err := v.getProvider(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	verifier := oidc.NewVerifier(provider.issuer, provider.keySet, &oidc.Config{SkipClientIDCheck: true})
	idToken, err := verifier.Verify(ctx, rawToken)
	if err != nil {
		return nil, ErrInvalidBearerToken
	}

	audiences := matchingAudiences(idToken.Audience, allowedAudiences)
	if len(audiences) == 0 {
		return nil, ErrInvalidBearerToken
	}

	var claims map[string]any
	if err := idToken.Claims(&claims); err != nil {
		return nil, ErrInvalidBearerToken
	}

	if idToken.Subject == "" {
		return nil, ErrInvalidBearerToken
	}

	return &verifiedToken{subject: idToken.Subject, audiences: audiences, claims: claims}, nil
}

func matchingAudiences(tokenAudiences []string, allowedAudiences []string) []string {
	allowed := make(map[string]struct{}, len(allowedAudiences))
	for _, audience := range allowedAudiences {
		allowed[audience] = struct{}{}
	}

	matches := make([]string, 0, len(tokenAudiences))
	for _, audience := range tokenAudiences {
		if _, ok := allowed[audience]; !ok {
			continue
		}

		matches = append(matches, audience)
		delete(allowed, audience)
	}

	return matches
}

func (v *oidcTokenVerifier) getProvider(ctx context.Context, endpoint string) (*providerState, error) {
	if provider, ok := v.providers.GetIfPresent(endpoint); ok {
		return provider, nil
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	result := v.providerLoader.DoChan(endpoint, func() (any, error) {
		// another load may have populated the cache between the first lookup
		// and joining this singleflight call.
		if provider, ok := v.providers.GetIfPresent(endpoint); ok {
			return provider, nil
		}

		// discovery is shared by all requests for this endpoint and therefore
		// must not inherit cancellation from whichever request arrived first.
		provider, err := v.createProvider(context.Background(), endpoint)
		if err != nil {
			return nil, err
		}

		v.providers.Set(endpoint, provider)
		return provider, nil
	})

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-result:
		if result.Err != nil {
			return nil, result.Err
		}

		provider, ok := result.Val.(*providerState)
		if !ok {
			return nil, errors.New("unexpected OIDC provider cache value")
		}
		return provider, nil
	}
}

func (v *oidcTokenVerifier) createProvider(ctx context.Context, endpoint string) (*providerState, error) {
	if err := validateHTTPSURL(endpoint); err != nil {
		return nil, fmt.Errorf("invalid OIDC discovery endpoint: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
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
	return &providerState{
		issuer: document.Issuer,
		keySet: oidc.NewRemoteKeySet(keySetContext, document.JWKSURI),
	}, nil
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
