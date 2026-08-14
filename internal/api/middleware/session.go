package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/konfidence-project/konfidence/internal/api/apierror"
	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	"github.com/konfidence-project/konfidence/internal/api/session"
	authdomain "github.com/konfidence-project/konfidence/internal/auth"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"
)

func SessionAuthentication(logger *slog.Logger, store session.Reader, authRepo authdomain.Repository,
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
		logger: logger, store: store, authRepo: authRepo, cookieName: cfg.Session.Cookie.Name,
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

type sessionAuthenticator struct {
	logger     *slog.Logger
	store      session.Reader
	authRepo   authdomain.Repository
	cookieName string
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
		return fmt.Errorf("no matching session found")
	}

	projectRoles, err := a.authRepo.GetProjectRoles(ctx, storedSession.Groups)
	if err != nil {
		return err
	}
	storedSession.ProjectRoles = projectRoles

	*r = *r.WithContext(session.NewContext(r.Context(), storedSession))
	return nil
}
