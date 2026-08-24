package oidc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type Config struct {
	IdentityProviderURI string
	TokenURL            string
	AuthorizationURL    string
	DeviceAuthURL       string
	UserInfoURL         string
	JWKSURL             string
	ClientID            string
	ClientSecret        string
	Scopes              []string
	RedirectURI         string
	PKCEEnabled         bool
}

type Client struct {
	config          Config
	oidcProvider    *oidc.Provider
	oauth2Config    oauth2.Config
	idTokenVerifier oidc.IDTokenVerifier
}

type RefreshResult struct {
	Token   *oauth2.Token
	Subject string
	Claims  IDTokenAdditionalClaims
}

type Refresher interface {
	Refresh(ctx context.Context, token *oauth2.Token) (*RefreshResult, error)
}

func NewOIDCClient(config Config) *Client {
	return &Client{config: config}
}

type StateData struct {
	State               string
	Nonce               string
	ReturnURL           string
	CodeVerifier        string
	CodeChallengeMethod string
	CodeChallenge       string
	ClientCodeChallenge string
	CreatedAt           time.Time
}

// TokenResponse represents the response from the token endpoint.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
	Expiry       time.Time
}

// IDTokenAdditionalClaims represents the claims in the ID token.
type IDTokenAdditionalClaims struct {
	Nbf               int64    `json:"nbf,omitempty"`
	Email             *string  `json:"email,omitempty"`
	Groups            []string `json:"groups,omitempty"`
	Name              *string  `json:"name,omitempty"`
	PreferredUsername *string  `json:"preferred_username,omitempty"`
	GivenName         *string  `json:"given_name,omitempty"`
	MiddleName        *string  `json:"middle_name,omitempty"`
	FamilyName        *string  `json:"family_name,omitempty"`
	JobTitle          *string  `json:"job_title,omitempty"`
}

func (c *Client) Setup(ctx context.Context) error {
	// TODO setup a custom provider if no base idp url has been provided
	oidcProvider, err := oidc.NewProvider(ctx, c.config.IdentityProviderURI)
	if err != nil {
		return fmt.Errorf("failed to create oidc provider: %w", err)
	}
	c.oidcProvider = oidcProvider

	endpoint := c.oidcProvider.Endpoint()
	if c.config.TokenURL != "" {
		endpoint.TokenURL = c.config.TokenURL
	}
	if c.config.AuthorizationURL != "" {
		endpoint.AuthURL = c.config.AuthorizationURL
	}
	if c.config.DeviceAuthURL != "" {
		endpoint.DeviceAuthURL = c.config.DeviceAuthURL
	}

	endpoint.AuthStyle = oauth2.AuthStyleInHeader

	c.oauth2Config = oauth2.Config{
		ClientID:     c.config.ClientID,
		ClientSecret: c.config.ClientSecret,
		RedirectURL:  c.config.RedirectURI,
		Endpoint:     endpoint,
		Scopes:       c.config.Scopes,
	}

	// create an ID Token verifier.
	c.idTokenVerifier = *c.oidcProvider.Verifier(&oidc.Config{ClientID: c.config.ClientID})
	return nil
}

// GenerateState generates a new state and optional PKCE parameters.
func (c *Client) GenerateState(returnURL string) (*StateData, error) {
	state, err := generateRandomString(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	nonce, err := generateRandomString(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	now := time.Now()
	data := &StateData{
		State:     state,
		Nonce:     nonce,
		ReturnURL: returnURL,
		CreatedAt: now,
	}

	// generate PKCE parameters if enabled
	if c.config.PKCEEnabled {
		data.CodeVerifier = oauth2.GenerateVerifier()
		data.CodeChallenge = oauth2.S256ChallengeFromVerifier(data.CodeVerifier)
		data.CodeChallengeMethod = "S256"
	}

	return data, nil
}

func (c *Client) AuthCodeURL(state *StateData) string {
	options := []oauth2.AuthCodeOption{
		oidc.Nonce(state.Nonce),
		oauth2.SetAuthURLParam("response_mode", "query"),
	}

	if c.config.PKCEEnabled {
		options = append(options, oauth2.S256ChallengeOption(state.CodeVerifier))
	}

	return c.oauth2Config.AuthCodeURL(state.State, options...)
}

func (c *Client) Exchange(ctx context.Context, code string, state *StateData) (*oauth2.Token, error) {
	var options []oauth2.AuthCodeOption
	if c.config.PKCEEnabled {
		options = append(options, oauth2.VerifierOption(state.CodeVerifier))
	}

	token, err := c.oauth2Config.Exchange(ctx, code, options...)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	return token, nil
}

func (c *Client) VerifyAndGetIdToken(ctx context.Context, token *oauth2.Token) (*oidc.IDToken, error) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("missing id token in token response")
	}

	idToken, err := c.idTokenVerifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify id token")
	}

	return idToken, nil
}

func (c *Client) GetClaims(userInformation *oidc.UserInfo) (*IDTokenAdditionalClaims, error) {
	var claims IDTokenAdditionalClaims
	if err := userInformation.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	return &claims, nil
}

func (c *Client) GetUserInformation(ctx context.Context, accessToken string) (*oidc.UserInfo, error) {
	token := &oauth2.Token{
		AccessToken: accessToken,
		TokenType:   "Bearer",
	}
	tokenSource := oauth2.StaticTokenSource(token)
	return c.oidcProvider.UserInfo(ctx, tokenSource)
}

func (c *Client) Refresh(ctx context.Context, token *oauth2.Token) (*RefreshResult, error) {
	if token == nil {
		return nil, fmt.Errorf("token refresh failed: token is empty")
	}

	// do token refresh
	refreshedToken, err := c.oauth2Config.TokenSource(ctx, token).Token()
	if err != nil {
		return nil, fmt.Errorf("token refresh failed: %w", err)
	}

	// some providers only return a refresh token when it is rotated.
	if refreshedToken.RefreshToken == "" {
		refreshedToken.RefreshToken = token.RefreshToken
	}

	userInformation, err := c.GetUserInformation(ctx, refreshedToken.AccessToken)
	if err != nil {
		return nil, fmt.Errorf(
			"getting user information after token refresh: %w",
			err,
		)
	}

	claims, err := c.GetClaims(userInformation)
	if err != nil {
		return nil, fmt.Errorf(
			"getting claims after token refresh: %w",
			err,
		)
	}

	return &RefreshResult{
		Token:   refreshedToken,
		Subject: userInformation.Subject,
		Claims:  *claims,
	}, nil
}

func generateRandomString(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
