package handler

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/konfidence-project/konfidence/internal/api/apierror"
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

func (a *authHandler) LoginV1(ctx context.Context, request openapi.LoginV1RequestObject) (openapi.LoginV1ResponseObject, error) {
	codeChallenge := ""
	if request.Params.CodeChallenge != nil {
		codeChallenge = *request.Params.CodeChallenge
	}

	if codeChallenge == "" {
		if !allowedReturnURL(request.Params.ReturnUrl, a.config.OIDC.AllowReturnURLs) {
			a.logger.WarnContext(ctx, "return URL is not allowed", "return_url", request.Params.ReturnUrl)
			return openapi.LoginV1400JSONResponse{
				BadRequestJSONResponse: apierror.NewBadRequestResponse("return URL is not allowed"),
			}, nil
		}
	} else if !allowedCLIReturnURL(request.Params.ReturnUrl) {
		a.logger.WarnContext(ctx, "CLI return URL is not a loopback URL", "return_url", request.Params.ReturnUrl)
		return openapi.LoginV1400JSONResponse{
			BadRequestJSONResponse: apierror.NewBadRequestResponse("return URL is not allowed"),
		}, nil
	}

	if !a.config.OIDC.Enabled {
		returnURL := request.Params.ReturnUrl
		return openapi.LoginV1302Response{
			Headers: openapi.LoginV1302ResponseHeaders{Location: &returnURL},
		}, nil
	}

	state, err := a.oidcClient.GenerateState(request.Params.ReturnUrl)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to generate oidc state", "error", err)
		return openapi.LoginV1500JSONResponse{
			InternalErrorJSONResponse: apierror.NewInternalErrorResponse(),
		}, nil
	}

	state.ClientCodeChallenge = codeChallenge

	if err := a.stateCache.Save(ctx, state); err != nil {
		a.logger.ErrorContext(ctx, "failed to save oidc state", "error", err)
		return openapi.LoginV1500JSONResponse{
			InternalErrorJSONResponse: apierror.NewInternalErrorResponse(),
		}, nil
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
		a.logger.WarnContext(ctx, "missing state parameter")
		return openapi.AuthCallbackV1400JSONResponse{
			BadRequestJSONResponse: apierror.NewBadRequestResponse("state parameter is required"),
		}, nil
	}

	storedState, err := a.stateCache.Consume(ctx, state)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to consume oidc state", "error", err)
		return openapi.AuthCallbackV1500JSONResponse{
			InternalErrorJSONResponse: apierror.NewInternalErrorResponse(),
		}, nil
	}
	if storedState == nil {
		a.logger.WarnContext(ctx, "oidc state is invalid or expired")
		return openapi.AuthCallbackV1400JSONResponse{
			BadRequestJSONResponse: apierror.NewBadRequestResponse("state is invalid or expired"),
		}, nil
	}

	// forward auth error to CLI callback
	if storedState.ClientCodeChallenge != "" && request.Params.Error != nil && *request.Params.Error != "" {
		callbackURL, err := url.Parse(storedState.ReturnURL)
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to parse CLI callback URL", "error", err)
			return openapi.AuthCallbackV1500JSONResponse{
				InternalErrorJSONResponse: apierror.NewInternalErrorResponse(),
			}, nil
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
		a.logger.WarnContext(ctx, "missing authorization code parameter")
		return openapi.AuthCallbackV1400JSONResponse{
			BadRequestJSONResponse: apierror.NewBadRequestResponse("authorization code parameter is required"),
		}, nil
	}

	// use auth code to get tokens
	tokenResponse, err := a.oidcClient.Exchange(ctx, *code, storedState)
	if err != nil {
		a.logger.ErrorContext(ctx, "token exchange failed", "error", err)
		return openapi.AuthCallbackV1400JSONResponse{
			BadRequestJSONResponse: apierror.NewBadRequestResponse("authorization code is invalid or expired"),
		}, nil
	}

	// verify and extract the id token
	idToken, err := a.oidcClient.VerifyAndGetIdToken(ctx, tokenResponse)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to verify or extract idToken", "error", err)
		return openapi.AuthCallbackV1500JSONResponse{
			InternalErrorJSONResponse: apierror.NewInternalErrorResponse(),
		}, nil
	}

	// check that nonce values match
	if storedState.Nonce != "" && storedState.Nonce != idToken.Nonce {
		a.logger.WarnContext(ctx, "idToken nonce does not match oidc state")
		return openapi.AuthCallbackV1400JSONResponse{
			BadRequestJSONResponse: apierror.NewBadRequestResponse("ID token nonce is invalid"),
		}, nil
	}

	userInformation, err := a.oidcClient.GetUserInformation(ctx, tokenResponse.AccessToken)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get user information", "error", err)
		return openapi.AuthCallbackV1401JSONResponse{
			UnauthorizedJSONResponse: apierror.NewUnauthorizedResponse(),
		}, nil
	}

	if userInformation.Subject != idToken.Subject {
		a.logger.ErrorContext(ctx, "user information subject does not match ID token subject")
		return openapi.AuthCallbackV1400JSONResponse{
			BadRequestJSONResponse: apierror.NewBadRequestResponse("user information does not match ID token"),
		}, nil
	}

	// extract additional claims
	claims, err := a.oidcClient.GetClaims(userInformation)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to parse additional claims from user information", "error", err)
		return openapi.AuthCallbackV1500JSONResponse{
			InternalErrorJSONResponse: apierror.NewInternalErrorResponse(),
		}, nil
	}

	var refreshToken *string
	if tokenResponse.RefreshToken != "" {
		refreshToken = &tokenResponse.RefreshToken
	}

	// create and save session
	sess := session.Session{}
	sess.ApplyOIDCValues(
		idToken.Subject,
		*claims,
		tokenResponse.AccessToken,
		refreshToken,
		tokenResponse.Expiry,
	)

	sessionId, err := a.sessions.Save(ctx, &sess)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to create session", "error", err)
		return openapi.AuthCallbackV1500JSONResponse{
			InternalErrorJSONResponse: apierror.NewInternalErrorResponse(),
		}, nil
	}

	sess.ID = sessionId

	if storedState.ClientCodeChallenge != "" {
		exchangeCode := oauth2.GenerateVerifier()
		if err := a.exchangeCache.Save(ctx, exchangeCode, oidc.Exchange{
			SessionID:     sessionId,
			CodeChallenge: storedState.ClientCodeChallenge,
		}); err != nil {
			a.logger.ErrorContext(ctx, "failed to save CLI exchange code", "error", err)
			return openapi.AuthCallbackV1500JSONResponse{
				InternalErrorJSONResponse: apierror.NewInternalErrorResponse(),
			}, nil
		}

		callbackURL, err := url.Parse(storedState.ReturnURL)
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to parse CLI callback URL", "error", err)
			return openapi.AuthCallbackV1500JSONResponse{
				InternalErrorJSONResponse: apierror.NewInternalErrorResponse(),
			}, nil
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
		a.logger.WarnContext(ctx, "session does not exist in context", "error", err)
		return openapi.LogoutV1401JSONResponse{
			UnauthorizedJSONResponse: apierror.NewUnauthorizedResponse(),
		}, nil
	}

	if err := a.sessions.Delete(ctx, storedSession.ID); err != nil {
		a.logger.ErrorContext(ctx, "failed to delete session", "error", err)
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
		a.logger.WarnContext(ctx, "session does not exist in context", "error", err)
		return openapi.GetIdentityV1401JSONResponse{
			UnauthorizedJSONResponse: apierror.NewUnauthorizedResponse(),
		}, nil
	}

	return openapi.GetIdentityV1200JSONResponse{
		Email:        lo.FromPtr(storedSession.Email),
		Name:         lo.FromPtr(storedSession.Name),
		GivenName:    lo.FromPtr(storedSession.GivenName),
		FamilyName:   lo.FromPtr(storedSession.FamilyName),
		ProjectRoles: storedSession.ProjectRoles,
	}, nil
}

func (a *authHandler) PostExchangeCodeV1(ctx context.Context,
	request openapi.PostExchangeCodeV1RequestObject) (openapi.PostExchangeCodeV1ResponseObject, error) {
	if request.Body == nil || request.Body.Code == "" || request.Body.Verifier == "" {
		a.logger.WarnContext(ctx, "CLI exchange request is incomplete")
		return openapi.PostExchangeCodeV1401JSONResponse{
			UnauthorizedJSONResponse: apierror.NewUnauthorizedResponse(),
		}, nil
	}

	exchange, err := a.exchangeCache.Consume(ctx, request.Body.Code)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to consume CLI exchange code", "error", err)
		return openapi.PostExchangeCodeV1500JSONResponse{
			InternalErrorJSONResponse: apierror.NewInternalErrorResponse(),
		}, nil
	}
	if exchange == nil {
		a.logger.WarnContext(ctx, "CLI exchange code is invalid or expired")
		return openapi.PostExchangeCodeV1401JSONResponse{
			UnauthorizedJSONResponse: apierror.NewUnauthorizedResponse(),
		}, nil
	}

	actualChallenge := oauth2.S256ChallengeFromVerifier(request.Body.Verifier)
	if subtle.ConstantTimeCompare(
		[]byte(actualChallenge),
		[]byte(exchange.CodeChallenge),
	) != 1 {
		a.logger.WarnContext(ctx, "CLI exchange code verifier is invalid")
		return openapi.PostExchangeCodeV1401JSONResponse{
			UnauthorizedJSONResponse: apierror.NewUnauthorizedResponse(),
		}, nil
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
		MaxAge:   int(a.config.Session.Expiration / time.Second),
		Expires:  time.Now().Add(a.config.Session.Expiration),
		HttpOnly: a.config.Session.Cookie.HTTPOnly,
		Secure:   a.config.Session.Cookie.Secure,
		SameSite: sameSiteMode(a.config.Session.Cookie.SameSite),
		Path:     "/",
	}
}
