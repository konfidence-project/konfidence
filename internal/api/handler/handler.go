package handler

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/konfidence-project/konfidence/internal/api/apierror"
	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/konfidence-project/konfidence/internal/api/middleware"
	"github.com/konfidence-project/konfidence/internal/api/oidc"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	"github.com/konfidence-project/konfidence/internal/api/session"
	authdomain "github.com/konfidence-project/konfidence/internal/auth"
	landscapedomain "github.com/konfidence-project/konfidence/internal/landscape"
	projectdomain "github.com/konfidence-project/konfidence/internal/project"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// APIHandler is the public handle used in tests.
type APIHandler = apiHandler

type apiHandler struct {
	authHandler
	projectHandler
}

var _ openapi.StrictServerInterface = (*apiHandler)(nil)

func NewAPIHandler(logger *slog.Logger, k8sClient client.Client, oidcClient oidc.Client,
	sessionStore session.Store, cfg config.Parsed) (http.Handler, error) {
	auth := newAuthHandler(logger, oidcClient, oidc.NewStateCacheStore(cfg), sessionStore, cfg)

	authRepo := authdomain.NewRepository(k8sClient)
	projectRepo := projectdomain.NewRepository(k8sClient)
	landscapeRepo := landscapedomain.NewRepository(k8sClient)

	project := newProjectHandler(projectRepo, landscapeRepo)
	api := &apiHandler{
		authHandler:    *auth,
		projectHandler: *project,
	}
	return middleware.SessionAuthentication(logger, sessionStore, authRepo, cfg, api.handler())
}

// NewAPIHandlerWithRepos constructs an APIHandler with injected repositories, used in tests.
func NewAPIHandlerWithRepos(projectRepo projectdomain.Repository, landscapeRepo landscapedomain.Repository) *APIHandler {
	return &apiHandler{
		projectHandler: *newProjectHandler(projectRepo, landscapeRepo),
	}
}

func (s *apiHandler) handler() http.Handler {
	errHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		if apiErr := apierror.As(err); apiErr != nil {
			apierror.Write(w, apiErr)
			return
		}
		apierror.WriteInternal(w)
	}

	apiRouter := chi.NewRouter()
	apiRouter.Mount("/api",
		openapi.Handler(openapi.NewStrictHandlerWithOptions(s, nil, openapi.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  errHandler,
			ResponseErrorHandlerFunc: errHandler,
		})))
	return apiRouter
}