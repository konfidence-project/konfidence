package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"golang.org/x/oauth2"
)

const sessionCookieName = "kden_session"

type Role string

const (
	RoleAdmin Role = "ADMIN"
	RoleDev   Role = "DEV"
	RolePM    Role = "PM"
)

type Identity struct {
	Subject       string `json:"subject"`
	Username      string `json:"username"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	EmailVerified bool   `json:"emailVerified"`
	Role          Role   `json:"role"`
}

type AuthConfig struct {
	AuthorizeURL string
	TokenURL     string
	UserInfoURL  string
	ClientID     string
	RedirectURI  string
	Scopes       []string
}

type tokenExchanger interface {
	Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error)
}

type oauth2Exchanger struct {
	config oauth2.Config
}

func (e oauth2Exchanger) Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	return e.config.Exchange(ctx, code, opts...)
}

type authState struct {
	CodeVerifier string
	ReturnTo     string
}

type Auth struct {
	mu         sync.RWMutex
	sessions   map[string]Identity
	states     map[string]authState
	exchanges  map[string]string
	config     AuthConfig
	exchanger  tokenExchanger
	httpClient *http.Client
}

func NewAuth(cfg AuthConfig) *Auth {
	oauthCfg := oauth2.Config{
		ClientID:    cfg.ClientID,
		RedirectURL: cfg.RedirectURI,
		Scopes:      cfg.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  cfg.AuthorizeURL,
			TokenURL: cfg.TokenURL,
		},
	}

	return &Auth{
		sessions:   map[string]Identity{},
		states:     map[string]authState{},
		exchanges:  map[string]string{},
		config:     cfg,
		exchanger:  oauth2Exchanger{config: oauthCfg},
		httpClient: http.DefaultClient,
	}
}

func (a *Auth) LoginStart(w http.ResponseWriter, r *http.Request) error {
	if err := a.validateConfigured(); err != nil {
		return err
	}

	state, err := randomURLSafe(32)
	if err != nil {
		return NewInternal(err)
	}
	verifier, err := randomURLSafe(64)
	if err != nil {
		return NewInternal(err)
	}
	returnTo := strings.TrimSpace(r.URL.Query().Get("returnTo"))

	a.mu.Lock()
	a.states[state] = authState{CodeVerifier: verifier, ReturnTo: returnTo}
	a.mu.Unlock()

	redirectURL, err := url.Parse(a.config.AuthorizeURL)
	if err != nil {
		return NewInternal(err)
	}
	query := redirectURL.Query()
	query.Set("response_type", "code")
	query.Set("client_id", a.config.ClientID)
	query.Set("redirect_uri", a.config.RedirectURI)
	query.Set("scope", strings.Join(a.config.Scopes, " "))
	query.Set("state", state)
	query.Set("code_challenge", codeChallenge(verifier))
	query.Set("code_challenge_method", "S256")
	redirectURL.RawQuery = query.Encode()

	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
	return nil
}

func (a *Auth) Callback(w http.ResponseWriter, r *http.Request) error {
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	if code == "" || state == "" {
		return NewBadRequest("authorization callback requires code and state", nil)
	}

	a.mu.Lock()
	storedState, ok := a.states[state]
	delete(a.states, state)
	a.mu.Unlock()
	if !ok {
		return NewUnauthorized("invalid authorization state")
	}

	token, err := a.exchanger.Exchange(r.Context(), code, oauth2.SetAuthURLParam("code_verifier", storedState.CodeVerifier))
	if err != nil {
		return NewUnauthorized("authorization code exchange failed")
	}

	identity, err := a.fetchIdentity(r.Context(), token)
	if err != nil {
		return err
	}

	sessionID, err := randomURLSafe(32)
	if err != nil {
		return NewInternal(err)
	}
	exchangeCode, err := randomURLSafe(32)
	if err != nil {
		return NewInternal(err)
	}

	a.mu.Lock()
	a.sessions[sessionID] = identity
	a.exchanges[exchangeCode] = sessionID
	a.mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	if storedState.ReturnTo != "" {
		redirectURL, err := url.Parse(storedState.ReturnTo)
		if err != nil {
			return NewBadRequest("returnTo contains an invalid URL", err)
		}
		query := redirectURL.Query()
		query.Set("code", exchangeCode)
		redirectURL.RawQuery = query.Encode()
		http.Redirect(w, r, redirectURL.String(), http.StatusFound)
		return nil
	}

	writeJSON(w, http.StatusOK, struct {
		SessionID string `json:"sessionId"`
	}{SessionID: sessionID})
	return nil
}

type exchangeRequest struct {
	Code string `json:"code"`
}

func (a *Auth) Exchange(w http.ResponseWriter, r *http.Request) error {
	var req exchangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return NewBadRequest("session exchange requires a code", err)
	}
	code := strings.TrimSpace(req.Code)
	if code == "" {
		return NewBadRequest("session exchange requires a code", nil)
	}

	a.mu.Lock()
	sessionID, ok := a.exchanges[code]
	delete(a.exchanges, code)
	a.mu.Unlock()
	if !ok {
		return NewUnauthorized("invalid session exchange code")
	}

	writeJSON(w, http.StatusOK, struct {
		SessionID string `json:"sessionId"`
	}{SessionID: sessionID})
	return nil
}

func (a *Auth) RequireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionID := sessionIDFromRequest(r)
		if sessionID == "" {
			WriteAPIError(w, NewUnauthorized("missing session"))
			return
		}

		a.mu.RLock()
		_, ok := a.sessions[sessionID]
		a.mu.RUnlock()
		if !ok {
			WriteAPIError(w, NewUnauthorized("invalid session"))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (a *Auth) Logout(w http.ResponseWriter, r *http.Request) error {
	sessionID := sessionIDFromRequest(r)
	if sessionID == "" {
		return NewUnauthorized("missing session")
	}

	a.mu.Lock()
	delete(a.sessions, sessionID)
	a.mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	writeJSON(w, http.StatusOK, struct {
		Status string `json:"status"`
	}{Status: "logged_out"})
	return nil
}

func (a *Auth) Identity(w http.ResponseWriter, r *http.Request) error {
	sessionID := sessionIDFromRequest(r)
	if sessionID == "" {
		return NewUnauthorized("missing session")
	}

	a.mu.RLock()
	identity, ok := a.sessions[sessionID]
	a.mu.RUnlock()
	if !ok {
		return NewUnauthorized("invalid session")
	}

	writeJSON(w, http.StatusOK, identity)
	return nil
}

func (a *Auth) fetchIdentity(ctx context.Context, token *oauth2.Token) (Identity, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.config.UserInfoURL, nil)
	if err != nil {
		return Identity{}, NewInternal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return Identity{}, NewUnauthorized("userinfo request failed")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return Identity{}, NewUnauthorized("userinfo request failed")
	}

	var claims struct {
		Subject       string   `json:"sub"`
		PreferredName string   `json:"preferred_username"`
		Username      string   `json:"username"`
		Email         string   `json:"email"`
		Name          string   `json:"name"`
		EmailVerified bool     `json:"email_verified"`
		Groups        []string `json:"groups"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&claims); err != nil {
		return Identity{}, NewInternal(err)
	}

	return Identity{
		Subject:       claims.Subject,
		Username:      firstNonEmpty(claims.PreferredName, claims.Username, claims.Email, claims.Subject),
		Email:         claims.Email,
		Name:          claims.Name,
		EmailVerified: claims.EmailVerified,
		Role:          roleFromGroups(claims.Groups),
	}, nil
}

func (a *Auth) validateConfigured() error {
	if strings.TrimSpace(a.config.AuthorizeURL) == "" || strings.TrimSpace(a.config.TokenURL) == "" ||
		strings.TrimSpace(a.config.UserInfoURL) == "" || strings.TrimSpace(a.config.ClientID) == "" ||
		strings.TrimSpace(a.config.RedirectURI) == "" {
		return NewInternal(fmt.Errorf("auth idp is not configured"))
	}
	return nil
}

func codeChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func randomURLSafe(size int) (string, error) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func sessionIDFromRequest(r *http.Request) string {
	if cookie, err := r.Cookie(sessionCookieName); err == nil && strings.TrimSpace(cookie.Value) != "" {
		return strings.TrimSpace(cookie.Value)
	}
	if sessionID := strings.TrimSpace(r.Header.Get("X-Session-ID")); sessionID != "" {
		return sessionID
	}

	value := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(value), "bearer ") {
		return strings.TrimSpace(value[len("bearer "):])
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func roleFromGroups(groups []string) Role {
	for _, group := range groups {
		switch strings.ToLower(group) {
		case "admin", "admins", "konfidence-admins":
			return RoleAdmin
		case "pm", "product", "product-managers", "konfidence-pms":
			return RolePM
		}
	}
	return RoleDev
}
