package handler

import (
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

type ServerHandler struct {
	AuthHandler
	ProjectHandler
	authMiddleware func(http.Handler) http.Handler
}

var _ openapi.StrictServerInterface = (*ServerHandler)(nil)

func NewServerHandler(logger *slog.Logger, k8s client.Client, oidcClient oidc.Client,
	stateStore oidc.StateStore, sessionStore session.SessionStore, cfg config.Parsed) (*ServerHandler, error) {
	authHandler := NewAuthHandler(logger, oidcClient, stateStore, sessionStore, cfg, k8s)
	projectHandler := NewProjectHandler(k8s)
	authMiddleware, err := newSessionAuthMiddleware(authHandler)
	if err != nil {
		return nil, err
	}

	return &ServerHandler{
		AuthHandler:    *authHandler,
		ProjectHandler: *projectHandler,
		authMiddleware: authMiddleware,
	}, nil
}

func newSessionAuthMiddleware(authHandler *AuthHandler) (func(http.Handler) http.Handler, error) {
	spec, err := openapi.GetSpec()
	if err != nil {
		return nil, fmt.Errorf("loading OpenAPI spec: %w", err)
	}

	authMiddleware := nethttpmiddleware.OapiRequestValidatorWithOptions(spec, &nethttpmiddleware.Options{
		DoNotValidateServers: true,
		Prefix:               "/api/v1",
		Options: openapi3filter.Options{
			AuthenticationFunc: authHandler.AuthenticateSession,
		},
		ErrorHandler: func(w http.ResponseWriter, _ string, statusCode int) {
			http.Error(w, "", statusCode)
		},
	})

	return authMiddleware, nil
}

func (s *ServerHandler) Mount(r chi.Router) {
	errHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		if apiErr := AsAPIError(err); apiErr != nil {
			WriteAPIError(w, apiErr)
			return
		}
		WriteInternalError(w)
	}

	apiRouter := chi.NewRouter()
	openapi.HandlerWithOptions(openapi.NewStrictHandlerWithOptions(s, nil, openapi.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  errHandler,
		ResponseErrorHandlerFunc: errHandler,
	}), openapi.ChiServerOptions{
		BaseURL:    "/api/v1",
		BaseRouter: apiRouter,
	})
	r.Mount("/", s.authMiddleware(apiRouter))
}
