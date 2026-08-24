package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/konfidence-project/konfidence/internal/api/apierror"
	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/konfidence-project/konfidence/internal/api/oidc"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	"github.com/konfidence-project/konfidence/internal/api/session"
	authdomain "github.com/konfidence-project/konfidence/internal/auth"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"
	"golang.org/x/oauth2"
	"golang.org/x/sync/singleflight"
)

type sessionAuthenticator struct {
	logger            *slog.Logger
	store             session.Store
	refresher         oidc.Refresher
	authRepo          authdomain.Repository
	cookieName        string
	refreshEnabled    bool
	tokenRefreshGroup singleflight.Group
}

func SessionAuthentication(logger *slog.Logger, store session.Store, refresher oidc.Refresher,
	authRepo authdomain.Repository,
	cfg config.Parsed, next http.Handler) (http.Handler, error) {
	spec, err := openapi.GetSpec()
	if err != nil {
		return nil, fmt.Errorf("loading OpenAPI spec: %w", err)
	}
	securityScheme := spec.Components.SecuritySchemes["sessionCookie"]
	if securityScheme == nil || securityScheme.Value == nil {
		return nil, fmt.Errorf("sessionCookie security scheme is missing from OpenAPI spec")
	}
	securityScheme.Value.Name = cfg.Session.Cookie.Name

	authenticator := sessionAuthenticator{
		logger:     logger,
		store:      store,
		refresher:  refresher,
		authRepo:   authRepo,
		cookieName: cfg.Session.Cookie.Name,
		refreshEnabled: slices.Contains(
			cfg.OIDC.Scopes,
			"offline_access",
		),
	}
	validate := nethttpmiddleware.OapiRequestValidatorWithOptions(spec, &nethttpmiddleware.Options{
		DoNotValidateServers: true,
		Prefix:               "/api",
		Options: openapi3filter.Options{
			AuthenticationFunc: authenticator.authenticate,
		},
		ErrorHandler: func(w http.ResponseWriter, _ string, _ int) {
			apierror.Write(w, apierror.NewUnauthorized())
		},
	})
	return validate(next), nil
}

func (a *sessionAuthenticator) authenticate(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
	if input.SecuritySchemeName != "sessionCookie" {
		return fmt.Errorf("security scheme %q is not sessionCookie", input.SecuritySchemeName)
	}

	r := input.RequestValidationInput.Request
	cookie, err := r.Cookie(a.cookieName)
	if err != nil {
		return fmt.Errorf("getting session cookie: %w", err)
	}

	storedSession, err := a.store.Get(ctx, cookie.Value)
	if err != nil {
		a.logger.Error("failed to get session", "error", err)
		return fmt.Errorf("getting session: %w", err)
	}
	if storedSession == nil {
		return a.expireSession(ctx, cookie.Value, fmt.Errorf("no matching session found"))
	}

	// check if the access token has expired
	// value of 0 means token will never expire
	if !storedSession.IsTokenExpiryZero() && storedSession.TokenExpiry <= time.Now().Unix() {
		if !a.refreshEnabled {
			return a.expireSession(ctx, cookie.Value, fmt.Errorf(
				"session token expired and token refresh is disabled",
			))
		}

		storedSession, err = a.refreshSession(ctx, storedSession.ID)
		if err != nil {
			return err
		}
	}

	projectRoles, err := a.authRepo.GetProjectRoles(ctx, storedSession.Groups)
	if err != nil {
		return err
	}
	storedSession.ProjectRoles = projectRoles

	*r = *r.WithContext(session.NewContext(r.Context(), storedSession))
	return nil
}

func (a *sessionAuthenticator) refreshSession(ctx context.Context, sessionId string) (*session.Session, error) {
	updatedSession, err, _ := a.tokenRefreshGroup.Do(sessionId, func() (any, error) {
		storedSession, err := a.store.Get(ctx, sessionId)
		if err != nil {
			return nil, fmt.Errorf("getting session before refresh failed: %w", err)
		}
		if storedSession == nil {
			return nil, fmt.Errorf("no matching session found")
		}

		// another request may already have refreshed the session.
		if storedSession.IsTokenExpiryZero() || storedSession.TokenExpiry > time.Now().Unix() {
			return storedSession, nil
		}

		return a.refreshExpiredSession(ctx, sessionId, storedSession)
	})
	if err != nil {
		return nil, err
	}

	// each waiting request receives its own copy because authentication subsequently
	// assigns request-specific project roles.
	refreshed := *updatedSession.(*session.Session)
	return &refreshed, nil
}

func (a *sessionAuthenticator) refreshExpiredSession(ctx context.Context,
	id string, storedSession *session.Session) (*session.Session, error) {
	if storedSession.RefreshToken == nil || *storedSession.RefreshToken == "" {
		return nil, a.expireSession(ctx, id, fmt.Errorf("session token expired without a refresh token"))
	}

	refreshed, err := a.refresher.Refresh(ctx, &oauth2.Token{
		AccessToken:  storedSession.AccessToken,
		RefreshToken: *storedSession.RefreshToken,
		TokenType:    "Bearer",
		Expiry:       time.Unix(storedSession.TokenExpiry, 0),
	})
	if err != nil {
		return nil, a.expireSession(ctx, id, fmt.Errorf("refreshing session token failed: %w", err))
	}

	if refreshed == nil || refreshed.Token == nil {
		return nil, a.expireSession(ctx, id, fmt.Errorf("token refresh returned an empty response"))
	}

	if refreshed.Token.AccessToken == "" {
		return nil, a.expireSession(ctx, id, fmt.Errorf("token refresh returned an empty access token"))
	}

	if !refreshed.Token.Expiry.IsZero() && !refreshed.Token.Expiry.After(time.Now()) {
		return nil, a.expireSession(ctx, id, fmt.Errorf("token refresh returned an expired token"))
	}

	if refreshed.Subject != storedSession.Subject {
		return nil, a.expireSession(ctx, id, fmt.Errorf("refreshed identity subject does not match session"))
	}

	refreshToken := refreshed.Token.RefreshToken
	if refreshToken == "" {
		return nil, a.expireSession(ctx, id, fmt.Errorf("token refresh returned an empty refresh token"))
	}

	updated := *storedSession
	updated.ApplyOIDCValues(
		refreshed.Subject,
		refreshed.Claims,
		refreshed.Token.AccessToken,
		&refreshToken,
		refreshed.Token.Expiry,
	)

	if err := a.store.Update(ctx, &updated); err != nil {
		return nil, a.expireSession(ctx, id, fmt.Errorf("updating refreshed session failed: %w", err))
	}

	return &updated, nil
}

func (a *sessionAuthenticator) expireSession(ctx context.Context, id string, cause error) error {
	if err := a.store.Delete(ctx, id); err != nil {
		a.logger.Error("failed to delete expired session", "session_id", id, "error", err)
	}

	return cause
}
