package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/konfidence-project/konfidence/internal/api/oidc"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	"github.com/konfidence-project/konfidence/internal/api/session"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type APIHandler struct {
	AuthHandler
	ProjectHandler
	authMiddleware func(http.Handler) http.Handler
}

type sessionMiddleware struct {
	logger *slog.Logger
	store  session.SessionStore
	config config.Parsed
}

func (m *sessionMiddleware) AuthenticateSession(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
	if input.SecuritySchemeName != "sessionCookie" {
		return fmt.Errorf("security scheme %q is not sessionCookie", input.SecuritySchemeName)
	}

	r := input.RequestValidationInput.Request
	sessionCookie, err := r.Cookie(m.config.SessionCookieName)
	if err != nil {
		return fmt.Errorf("getting session cookie: %w", err)
	}

	storedSession, err := m.store.Get(ctx, sessionCookie.Value)
	if err != nil {
		m.logger.Error("failed to get session", "error", err)
		return fmt.Errorf("getting session: %w", err)
	}
	if storedSession == nil {
		return fmt.Errorf("no matching session found")
	}

	*r = *r.WithContext(session.NewContext(r.Context(), storedSession))
	return nil
}

var _ openapi.StrictServerInterface = (*APIHandler)(nil)

func NewAPIHandler(logger *slog.Logger, k8s client.Client, oidcClient oidc.Client,
	stateStore oidc.StateStore, sessionStore session.SessionStore, cfg config.Parsed) (*APIHandler, error) {
	sessions := &sessionMiddleware{logger: logger, store: sessionStore, config: cfg}
	authHandler := NewAuthHandler(logger, oidcClient, stateStore, sessionStore, cfg, k8s)
	projectHandler := NewProjectHandler(k8s)
	authMiddleware, err := newSessionAuthMiddleware(sessions)
	if err != nil {
		return nil, err
	}

	return &APIHandler{
		AuthHandler:    *authHandler,
		ProjectHandler: *projectHandler,
		authMiddleware: authMiddleware,
	}, nil
}

func newSessionAuthMiddleware(sessions *sessionMiddleware) (func(http.Handler) http.Handler, error) {
	spec, err := openapi.GetSpec()
	if err != nil {
		return nil, fmt.Errorf("loading OpenAPI spec: %w", err)
	}

	authMiddleware := nethttpmiddleware.OapiRequestValidatorWithOptions(spec, &nethttpmiddleware.Options{
		DoNotValidateServers: true,
		Prefix:               "/api",
		Options: openapi3filter.Options{
			AuthenticationFunc: sessions.AuthenticateSession,
		},
		ErrorHandler: func(w http.ResponseWriter, _ string, statusCode int) {
			http.Error(w, "", statusCode)
		},
	})

	return authMiddleware, nil
}

func (s *APIHandler) Handler() http.Handler {
	errHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		if apiErr := AsAPIError(err); apiErr != nil {
			WriteAPIError(w, apiErr)
			return
		}
		WriteInternalError(w)
	}

	apiRouter := chi.NewRouter()
	apiRouter.Mount("/api",
		openapi.Handler(openapi.NewStrictHandlerWithOptions(s, nil, openapi.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  errHandler,
			ResponseErrorHandlerFunc: errHandler,
		})))
	return s.authMiddleware(apiRouter)
}
