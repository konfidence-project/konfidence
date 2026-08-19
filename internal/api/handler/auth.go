package handler

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"slices"

	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/konfidence-project/konfidence/internal/api/oidc"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	"github.com/konfidence-project/konfidence/internal/api/session"
	"github.com/samber/lo"
	"golang.org/x/oauth2"
)

type authHandler struct {
	logger        *slog.Logger
	oidcClient    oidc.Client
	stateCache    oidc.StateStore
	exchangeCache oidc.ExchangeStore
	sessions      session.Writer
	config        config.Parsed
}

func newAuthHandler(logger *slog.Logger, oidcClient oidc.Client, stateStore oidc.StateStore, exchangeStore oidc.ExchangeStore,
	sessions session.Writer, cfg config.Parsed) *authHandler {
	return &authHandler{
		logger:        logger,
		oidcClient:    oidcClient,
		stateCache:    stateStore,
		exchangeCache: exchangeStore,
		sessions:      sessions,
		config:        cfg,
	}
}

func (a *authHandler) LoginV1(_ context.Context, request openapi.LoginV1RequestObject) (openapi.LoginV1ResponseObject, error) {
	codeChallenge := ""
	if request.Params.CodeChallenge != nil {
		codeChallenge = *request.Params.CodeChallenge
	}

	if codeChallenge == "" {
		if !allowedReturnURL(request.Params.ReturnUrl, a.config.OIDC.AllowReturnURLs) {
			a.logger.Warn("return URL is not allowed", "return_url", request.Params.ReturnUrl)
			return openapi.LoginV1400JSONResponse{}, nil
		}
	} else if !allowedCLIReturnURL(request.Params.ReturnUrl) {
		a.logger.Warn("CLI return URL is not a loopback URL", "return_url", request.Params.ReturnUrl)
		return openapi.LoginV1400JSONResponse{}, nil
	}

	state, err := a.oidcClient.GenerateState(request.Params.ReturnUrl)
	if err != nil {
		return nil, err
	}

	state.ClientCodeChallenge = codeChallenge

	if err := a.stateCache.Save(state); err != nil {
		return nil, err
	}

	authCodeURL := a.oidcClient.AuthCodeURL(state)
	return openapi.LoginV1302Response{
		Headers: openapi.LoginV1302ResponseHeaders{
			Location: &authCodeURL,
		},
	}, nil
}

func allowedReturnURL(returnURL string, allowReturnURLs []string) bool {
	return slices.Contains(allowReturnURLs, returnURL)
}

func allowedCLIReturnURL(rawURL string) bool {
	callbackURL, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	if callbackURL.Scheme != "http" || callbackURL.Path != "/callback" {
		return false
	}

	host := callbackURL.Hostname()
	if host != "127.0.0.1" && host != "::1" {
		return false
	}

	port := callbackURL.Port()
	if port == "" {
		return false
	}

	return net.ParseIP(host) != nil
}

func (a *authHandler) AuthCallbackV1(ctx context.Context, request openapi.AuthCallbackV1RequestObject) (openapi.AuthCallbackV1ResponseObject, error) {
	state := request.Params.State
	if state == "" {
		a.logger.Error("missing state parameter")
		return openapi.AuthCallbackV1400JSONResponse{}, nil
	}

	storedState, err := a.stateCache.Consume(state)
	if err != nil {
		return openapi.AuthCallbackV1500JSONResponse{}, nil
	}
	if storedState == nil {
		return openapi.AuthCallbackV1400JSONResponse{}, nil
	}

	// forward auth error to CLI callback
	if storedState.ClientCodeChallenge != "" && request.Params.Error != nil && *request.Params.Error != "" {
		callbackURL, err := url.Parse(storedState.ReturnURL)
		if err != nil {
			return nil, fmt.Errorf("parsing login return URL: %w", err)
		}

		query := callbackURL.Query()
		query.Set("error", *request.Params.Error)

		if request.Params.ErrorDescription != nil {
			query.Set("error_description", *request.Params.ErrorDescription)
		}

		callbackURL.RawQuery = query.Encode()
		location := callbackURL.String()

		return openapi.AuthCallbackV1302Response{
			Headers: openapi.AuthCallbackV1302ResponseHeaders{
				Location: &location,
			},
		}, nil
	}

	code := request.Params.Code
	if code == nil || *code == "" {
		a.logger.Error("missing authorization code parameter")
		return openapi.AuthCallbackV1400JSONResponse{}, nil
	}

	// use auth code to get tokens
	tokenResponse, err := a.oidcClient.Exchange(ctx, *code, storedState)
	if err != nil {
		a.logger.Error("token exchange failed", "error", err)
		return openapi.AuthCallbackV1400JSONResponse{}, nil
	}

	// verify and extract the id token
	idToken, err := a.oidcClient.VerifyAndGetIdToken(ctx, tokenResponse)
	if err != nil {
		a.logger.Error("failed to verify or extract idToken")
		return openapi.AuthCallbackV1500JSONResponse{}, err
	}

	// check that nonce values match
	if storedState.Nonce != "" && storedState.Nonce != idToken.Nonce {
		a.logger.Error("invalid nonce value in idToken")
		return openapi.AuthCallbackV1400JSONResponse{}, nil
	}

	userInformation, err := a.oidcClient.GetUserInformation(ctx, tokenResponse.AccessToken)
	if err != nil {
		a.logger.Error("failed to get user information")
		return openapi.AuthCallbackV1401JSONResponse{}, err
	}

	if userInformation.Subject != idToken.Subject {
		a.logger.Error("user information subject and token subject do not match. invalid idp configuration")
		return openapi.AuthCallbackV1400JSONResponse{}, err
	}

	// extract additional claims
	claims, err := a.oidcClient.GetClaims(userInformation)
	if err != nil {
		a.logger.Error("failed to parse idToken additional claims from user information")
		return openapi.AuthCallbackV1500JSONResponse{}, err
	}

	// create and save session
	sess := session.Session{
		Subject: idToken.Subject,
		Groups:  claims.Groups,
		Context: session.Context{
			Name:              claims.Name,
			GivenName:         claims.GivenName,
			FamilyName:        claims.FamilyName,
			PreferredUsername: claims.PreferredUsername,
			Email:             claims.Email,
		},
		AccessToken:  tokenResponse.AccessToken,
		RefreshToken: &tokenResponse.RefreshToken,
		Expiry:       tokenResponse.Expiry.Unix(),
	}

	sessionId, err := a.sessions.Save(ctx, &sess)
	if err != nil {
		a.logger.Error("failed to create session")
		return openapi.AuthCallbackV1500JSONResponse{}, err
	}

	sess.ID = sessionId

	if storedState.ClientCodeChallenge != "" {
		exchangeCode := oauth2.GenerateVerifier()
		if err := a.exchangeCache.Save(exchangeCode, oidc.Exchange{
			SessionID:     sessionId,
			CodeChallenge: storedState.ClientCodeChallenge,
		}); err != nil {
			return nil, err
		}

		callbackURL, err := url.Parse(storedState.ReturnURL)
		if err != nil {
			return nil, fmt.Errorf("parsing CLI callback URL failed: %w", err)
		}

		query := callbackURL.Query()
		query.Set("code", exchangeCode)
		callbackURL.RawQuery = query.Encode()
		location := callbackURL.String()

		return openapi.AuthCallbackV1302Response{
			Headers: openapi.AuthCallbackV1302ResponseHeaders{
				Location: &location,
			},
		}, nil
	}

	cookie := a.sessionCookie(sessionId)
	cookieValue := cookie.String()
	return openapi.AuthCallbackV1302Response{
		Headers: openapi.AuthCallbackV1302ResponseHeaders{
			Location:  &storedState.ReturnURL,
			SetCookie: &cookieValue,
		},
	}, nil
}

func (a *authHandler) LogoutV1(ctx context.Context, _ openapi.LogoutV1RequestObject) (openapi.LogoutV1ResponseObject, error) {
	// delete session
	storedSession, err := session.FromContext(ctx)
	if err != nil {
		a.logger.Error("failed to get session from context", "error", err)
		return openapi.LogoutV1401JSONResponse{}, nil
	}

	if err := a.sessions.Delete(ctx, storedSession.ID); err != nil {
		a.logger.Error("failed to delete session", "error", err)
	}

	// clear session cookie
	sessionCookie := &http.Cookie{
		Name:     a.config.Session.Cookie.Name,
		Value:    "",
		MaxAge:   -1,
		HttpOnly: a.config.Session.Cookie.HTTPOnly,
		Secure:   a.config.Session.Cookie.Secure,
		SameSite: sameSiteMode(a.config.Session.Cookie.SameSite),
		Path:     "/",
	}

	sCookieStr := sessionCookie.String()
	return openapi.LogoutV1200Response{
		Headers: openapi.LogoutV1200ResponseHeaders{
			SetCookie: &sCookieStr,
		},
	}, nil
}

func (a *authHandler) GetIdentityV1(ctx context.Context, _ openapi.GetIdentityV1RequestObject) (openapi.GetIdentityV1ResponseObject, error) {
	storedSession, err := session.FromContext(ctx)
	if err != nil {
		a.logger.Error("session does not exist in context")
		return openapi.GetIdentityV1401JSONResponse{}, nil
	}

	return openapi.GetIdentityV1200JSONResponse{
		Email:        lo.FromPtr(storedSession.Email),
		Name:         lo.FromPtr(storedSession.Name),
		GivenName:    lo.FromPtr(storedSession.GivenName),
		FamilyName:   lo.FromPtr(storedSession.FamilyName),
		ProjectRoles: storedSession.ProjectRoles,
	}, nil
}

func (a *authHandler) PostExchangeCodeV1(_ context.Context, request openapi.PostExchangeCodeV1RequestObject) (openapi.PostExchangeCodeV1ResponseObject, error) {
	if request.Body == nil || request.Body.Code == "" || request.Body.Verifier == "" {
		return openapi.PostExchangeCodeV1401JSONResponse{}, nil
	}

	exchange, err := a.exchangeCache.Consume(request.Body.Code)
	// TODO maybe throw 400 or 500 here
	if err != nil {
		return nil, err
	}
	if exchange == nil {
		return openapi.PostExchangeCodeV1401JSONResponse{}, nil
	}

	actualChallenge := oauth2.S256ChallengeFromVerifier(request.Body.Verifier)
	if subtle.ConstantTimeCompare(
		[]byte(actualChallenge),
		[]byte(exchange.CodeChallenge),
	) != 1 {
		return openapi.PostExchangeCodeV1401JSONResponse{}, nil
	}

	cookieValue := a.sessionCookie(exchange.SessionID).String()
	return openapi.PostExchangeCodeV1200Response{
		Headers: openapi.PostExchangeCodeV1200ResponseHeaders{
			SetCookie: &cookieValue,
		},
	}, nil
}

func sameSiteMode(mode string) http.SameSite {
	switch mode {
	case "SameSiteDefaultMode", "Default":
		return http.SameSiteDefaultMode
	case "SameSiteLaxMode", "Lax":
		return http.SameSiteLaxMode
	case "SameSiteNoneMode", "None":
		return http.SameSiteNoneMode
	case "SameSiteStrictMode", "Strict":
		return http.SameSiteStrictMode
	}
	return http.SameSiteDefaultMode
}

func (a *authHandler) sessionCookie(sessionID string) *http.Cookie {
	return &http.Cookie{
		Name:     a.config.Session.Cookie.Name,
		Value:    sessionID,
		HttpOnly: a.config.Session.Cookie.HTTPOnly,
		Secure:   a.config.Session.Cookie.Secure,
		SameSite: sameSiteMode(a.config.Session.Cookie.SameSite),
		Path:     "/",
	}
}
