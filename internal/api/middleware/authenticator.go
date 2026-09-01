package middleware

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/konfidence-project/konfidence/internal/api/apierror"
	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	"github.com/konfidence-project/konfidence/internal/api/session"
	authdomain "github.com/konfidence-project/konfidence/internal/auth"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"
)

type authenticator struct {
	logger              *slog.Logger
	store               session.Store
	authRepo            authdomain.Repository
	cookieName          string
	expireOnTokenExpiry bool
}

func Authenticator(logger *slog.Logger, store session.Store, authRepo authdomain.Repository,
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

	bearerScheme := spec.Components.SecuritySchemes["bearerAuth"]
	if bearerScheme == nil || bearerScheme.Value == nil {
		return nil, errors.New(
			"bearerAuth security scheme is missing from OpenAPI spec",
		)
	}

	authenticator := authenticator{
		logger:     logger,
		store:      store,
		authRepo:   authRepo,
		cookieName: cfg.Session.Cookie.Name,
		expireOnTokenExpiry: slices.Contains(
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
		ErrorHandlerWithOpts: func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request, opts nethttpmiddleware.ErrorHandlerOpts) {
			logger.InfoContext(ctx, fmt.Sprintf("openapi request validation failed: %s", err))

			authorization := r.Header.Get("Authorization")
			scheme, _, _ := strings.Cut(authorization, " ")
			if opts.StatusCode == http.StatusUnauthorized && strings.EqualFold(scheme, "Bearer") {
				w.Header().Set("WWW-Authenticate", "Bearer")
			}

			apierror.Write(w, &apierror.Error{
				Status:  opts.StatusCode,
				Code:    fmt.Sprintf("%d", opts.StatusCode),
				Message: http.StatusText(opts.StatusCode),
			})
		},
	})
	return validate(next), nil
}

func (a *authenticator) authenticate(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
	switch input.SecuritySchemeName {
	case "sessionCookie":
		return a.authenticateSession(ctx, input)
	case "bearerAuth":
		return a.authenticateBearer(ctx, input)
	default:
		return fmt.Errorf("unsupported security scheme %q", input.SecuritySchemeName)
	}
}

func (a *authenticator) authenticateSession(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
	request := input.RequestValidationInput.Request

	// a supplied Authorization header must not silently fall back to a cookie.
	if request.Header.Get("Authorization") != "" {
		return errors.New("authorization header requires bearer authentication")
	}

	cookie, err := request.Cookie(a.cookieName)
	if err != nil {
		return fmt.Errorf("getting session cookie: %w", err)
	}

	storedSession, err := a.store.Get(ctx, cookie.Value)
	if err != nil {
		a.logger.Error("failed to get session", "error", err)
		return fmt.Errorf("getting session: %w", err)
	}

	if storedSession == nil {
		return a.expireSession(ctx, cookie.Value, errors.New("no matching session found"))
	}

	if a.expireOnTokenExpiry && !storedSession.IsTokenExpiryZero() && storedSession.TokenExpiry <= time.Now().Unix() {
		return a.expireSession(ctx, cookie.Value, errors.New("session token expired"))
	}

	projectRoles, err := a.authRepo.GetProjectRoles(ctx, storedSession.Groups)
	if err != nil {
		return err
	}

	storedSession.ProjectRoles = projectRoles
	*request = *request.WithContext(session.NewContext(request.Context(), storedSession))
	return nil
}

func (a *authenticator) authenticateBearer(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
	request := input.RequestValidationInput.Request
	rawToken, err := parseBearerToken(request.Header.Get("Authorization"))
	if err != nil {
		return err
	}

	identity, err := a.authRepo.AuthenticateToken(ctx, rawToken)
	if err != nil {
		return fmt.Errorf("bearer token authentication failed: %w", err)
	}

	if identity == nil {
		return errors.New("bearer token authentication returned no identity")
	}

	requestIdentity := session.Context{Subject: identity.Subject, ProjectRoles: identity.ProjectRoles}
	*request = *request.WithContext(session.NewRequestContext(request.Context(), requestIdentity))
	return nil
}

func parseBearerToken(header string) (string, error) {
	scheme, token, found := strings.Cut(strings.TrimSpace(header), " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return "", errors.New("invalid bearer authorization header")
	}

	token = strings.TrimSpace(token)
	if token == "" || strings.ContainsAny(token, " \t\r\n") {
		return "", errors.New("invalid bearer authorization header")
	}

	return token, nil
}

func (a *authenticator) expireSession(ctx context.Context, id string, cause error) error {
	if err := a.store.Delete(ctx, id); err != nil {
		a.logger.Error("failed to delete expired session", "session_id", id, "error", err)
	}

	return cause
}
