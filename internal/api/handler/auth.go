package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/konfidence-project/konfidence/internal/api/oidc"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	"github.com/konfidence-project/konfidence/internal/api/session"
	"github.com/samber/lo"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type AuthHandler struct {
	logger     *slog.Logger
	oidcClient oidc.Client
	stateCache oidc.StateStore
	sessions   session.SessionStore
	config     config.Parsed
	k8s        client.Client
}

func NewAuthHandler(logger *slog.Logger, oidcClient oidc.Client,
	stateStore oidc.StateStore, sessions session.SessionStore, cfg config.Parsed, k8s client.Client) *AuthHandler {
	return &AuthHandler{
		logger:     logger,
		oidcClient: oidcClient,
		stateCache: stateStore,
		sessions:   sessions,
		config:     cfg,
		k8s:        k8s,
	}
}

// TODO check error handling in all implemented methods

func (a *AuthHandler) LoginV1(_ context.Context, request openapi.LoginV1RequestObject) (openapi.LoginV1ResponseObject, error) {
	returnUrl := "/"
	// TODO need to validate returnUrl parameter
	if request.Params.ReturnUrl != nil && *request.Params.ReturnUrl != "" {
		returnUrl = *request.Params.ReturnUrl
	}

	state, err := a.oidcClient.GenerateState(returnUrl)
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

func (a *AuthHandler) AuthCallbackV1(ctx context.Context, request openapi.AuthCallbackV1RequestObject) (openapi.AuthCallbackV1ResponseObject, error) {
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
		return openapi.AuthCallbackV1401JSONResponse{}, nil
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

	// TODO determine roles, for now use some default ones
	roles := []string{"admin", "dev"}

	// create and save session
	// TODO encrypt session id
	sess := session.Session{
		Subject:           idToken.Subject,
		Name:              claims.Name,
		GivenName:         claims.GivenName,
		FamilyName:        claims.FamilyName,
		PreferredUsername: claims.PreferredUsername,
		Email:             claims.Email,
		Groups:            claims.Groups,
		Roles:             roles,
		AccessToken:       tokenResponse.AccessToken,
		RefreshToken:      &tokenResponse.RefreshToken,
		Expiry:            tokenResponse.Expiry.Unix(),
	}

	sessionId, err := a.sessions.Save(ctx, &sess)
	if err != nil {
		a.logger.Error("failed to create session")
		return openapi.AuthCallbackV1500JSONResponse{}, err
	}

	sessionCookie := &http.Cookie{
		Name:     a.config.SessionCookieName,
		Value:    sessionId,
		HttpOnly: a.config.SessionCookieHTTPOnly,
		Secure:   a.config.SessionCookieSecure,
		SameSite: sameSiteMode(a.config.SessionCookieSameSite),
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

func (a *AuthHandler) LogoutV1(ctx context.Context, _ openapi.LogoutV1RequestObject) (openapi.LogoutV1ResponseObject, error) {
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
		Name:     a.config.SessionCookieName,
		Value:    "",
		MaxAge:   -1,
		HttpOnly: a.config.SessionCookieHTTPOnly,
		Secure:   a.config.SessionCookieSecure,
		SameSite: sameSiteMode(a.config.SessionCookieSameSite),
		Path:     "/",
	}

	sCookieStr := sessionCookie.String()
	return openapi.LogoutV1200Response{
		Headers: openapi.LogoutV1200ResponseHeaders{
			SetCookie: &sCookieStr,
		},
	}, nil
}

func (a *AuthHandler) GetIdentityV1(ctx context.Context, _ openapi.GetIdentityV1RequestObject) (openapi.GetIdentityV1ResponseObject, error) {
	storedSession, err := session.FromContext(ctx)
	if err != nil {
		a.logger.Error("session does not exist in context")
		return openapi.GetIdentityV1401JSONResponse{}, nil
	}

	return openapi.GetIdentityV1200JSONResponse{
		Email:      lo.FromPtr(storedSession.Email),
		Name:       lo.FromPtr(storedSession.Name),
		GivenName:  lo.FromPtr(storedSession.GivenName),
		FamilyName: lo.FromPtr(storedSession.FamilyName),
		Roles:      storedSession.Roles,
	}, nil
}

func (a *AuthHandler) PostExchangeCodeV1(_ context.Context, _ openapi.PostExchangeCodeV1RequestObject) (openapi.PostExchangeCodeV1ResponseObject, error) {
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
