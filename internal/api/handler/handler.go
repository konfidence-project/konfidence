package handler

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/konfidence-project/konfidence/internal/api/oidc"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	"github.com/konfidence-project/konfidence/internal/api/session"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ServerHandler struct {
	AuthHandler
	ProjectHandler
}

var _ openapi.StrictServerInterface = (*ServerHandler)(nil)

func NewServerHandler(logger *slog.Logger, k8s client.Client, oidcClient oidc.Client,
	stateStore oidc.StateStore, sessionStore session.SessionStore, cfg config.Parsed) (*ServerHandler, error) {
	authHandler := NewAuthHandler(logger, oidcClient, stateStore, sessionStore, cfg, k8s)
	projectHandler := NewProjectHandler(k8s)

	return &ServerHandler{
		*authHandler,
		*projectHandler,
	}, nil
}

func (s *ServerHandler) Mount(r chi.Router) {
	errHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		if apiErr := AsAPIError(err); apiErr != nil {
			WriteAPIError(w, apiErr)
			return
		}
		WriteInternalError(w)
	}

	strictMiddlewares := []openapi.StrictMiddlewareFunc{
		s.SessionAuthMiddleware,
	}

	openapi.HandlerWithOptions(openapi.NewStrictHandlerWithOptions(s, strictMiddlewares, openapi.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  errHandler,
		ResponseErrorHandlerFunc: errHandler,
	}), openapi.ChiServerOptions{
		BaseURL:    "/api/v1",
		BaseRouter: r,
	})
}
