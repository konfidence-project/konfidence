package handler

import (
	"context"
	"log/slog"
	"net/http"
	"slices"

	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/konfidence-project/konfidence/internal/api/oidc"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	"github.com/konfidence-project/konfidence/internal/api/session"
	"github.com/samber/lo"
)

type authHandler struct {
	logger     *slog.Logger
	oidcClient oidc.Client
	stateCache oidc.StateStore
	sessions   session.Writer
	config     config.Parsed
}

func newAuthHandler(logger *slog.Logger, oidcClient oidc.Client, stateStore oidc.StateStore, sessions session.Writer, cfg config.Parsed) *authHandler {
	return &authHandler{
		logger:     logger,
		oidcClient: oidcClient,
		stateCache: stateStore,
		sessions:   sessions,
		config:     cfg,
	}
}

func (a *authHandler) LoginV1(_ context.Context, request openapi.LoginV1RequestObject) (openapi.LoginV1ResponseObject, error) {
	if !allowedReturnURL(request.Params.ReturnUrl, a.config.OIDC.AllowReturnURLs) {
		a.logger.Warn("return URL is not allowed", "return_url", request.Params.ReturnUrl)
		return openapi.LoginV1400JSONResponse{}, nil
	}

	state, err := a.oidcClient.GenerateState(request.Params.ReturnUrl)
	if err != nil {
		return nil, err
	}

	err = a.stateCache.Save(state)
	if err != nil {
		return nil, err
	}

	// generate redirect url
	authCodeUrl := a.oidcClient.AuthCodeURL(state)
	return openapi.LoginV1302Response{
		Headers: openapi.LoginV1302ResponseHeaders{Location: &authCodeUrl},
	}, nil
}

func allowedReturnURL(returnURL string, allowReturnURLs []string) bool {
	return slices.Contains(allowReturnURLs, returnURL)
}

func (a *authHandler) AuthCallbackV1(ctx context.Context, request openapi.AuthCallbackV1RequestObject) (openapi.AuthCallbackV1ResponseObject, error) {
	state := request.Params.State
	if state == "" {
		a.logger.Error("missing state parameter")
		return openapi.AuthCallbackV1400JSONResponse{}, nil
	}

	code := request.Params.Code
	if code == "" {
		a.logger.Error("missing authorization code parameter")
		return openapi.AuthCallbackV1400JSONResponse{}, nil
	}

	storedState, err := a.stateCache.Get(state)
	if err != nil {
		a.logger.Error("failed to get stored oidc state", "error", err)
		return openapi.AuthCallbackV1500JSONResponse{}, nil
	}

	// entry does not exist or has been invalidated
	if storedState == nil {
		a.logger.Error("oidc state does not exist or has expired")
		return openapi.AuthCallbackV1400JSONResponse{}, nil
	}

	err = a.stateCache.Delete(storedState)
	if err != nil {
		a.logger.Error("failed to get delete stored oidc state", "error", err)
		// no error thrown here since stateStore should remove entry after TTL
	}

	// use auth code to get tokens
	tokenResponse, err := a.oidcClient.Exchange(ctx, code, storedState)
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
	sessionCookie := &http.Cookie{
		Name:     a.config.Session.Cookie.Name,
		Value:    sessionId,
		HttpOnly: a.config.Session.Cookie.HTTPOnly,
		Secure:   a.config.Session.Cookie.Secure,
		SameSite: sameSiteMode(a.config.Session.Cookie.SameSite),
		Path:     "/",
	}

	sCookieStr := sessionCookie.String()
	return openapi.AuthCallbackV1302Response{
		Headers: openapi.AuthCallbackV1302ResponseHeaders{
			Location:  &storedState.ReturnURL,
			SetCookie: &sCookieStr,
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

func (a *authHandler) PostExchangeCodeV1(_ context.Context, _ openapi.PostExchangeCodeV1RequestObject) (openapi.PostExchangeCodeV1ResponseObject, error) {
	return nil, nil
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
