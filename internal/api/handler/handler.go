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
	artifactdeploymentdomain "github.com/konfidence-project/konfidence/internal/artifactdeployment"
	authdomain "github.com/konfidence-project/konfidence/internal/auth"
	landscapedomain "github.com/konfidence-project/konfidence/internal/landscape"
	projectdomain "github.com/konfidence-project/konfidence/internal/project"
	stagedomain "github.com/konfidence-project/konfidence/internal/stage"
	vectordeploymentdomain "github.com/konfidence-project/konfidence/internal/vectordeployment"
	vectorpromotiondomain "github.com/konfidence-project/konfidence/internal/vectorpromotion"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type apiHandler struct {
	authHandler
	projectHandler
}

var _ openapi.StrictServerInterface = (*apiHandler)(nil)

func NewAPIHandler(logger *slog.Logger, k8sClient client.Client, oidcClient oidc.Client,
	stateStore oidc.StateStore, exchangeStore oidc.ExchangeStore, sessionStore session.Store,
	cfg config.Parsed,
) (http.Handler, error) {
	auth := newAuthHandler(logger, oidcClient, stateStore, exchangeStore, sessionStore, cfg)
	authRepo := authdomain.NewRepository(k8sClient)
	projectRepo := projectdomain.NewRepository(k8sClient)
	landscapeRepo := landscapedomain.NewRepository(k8sClient)
	stageRepo := stagedomain.NewRepository(k8sClient)
	artifactDeploymentRepo := artifactdeploymentdomain.NewRepository(k8sClient)
	vectorPromotionRepo := vectorpromotiondomain.NewRepository(k8sClient)
	vectorPromotionConfigRepo := vectorpromotiondomain.NewConfigRepository(k8sClient)
	vectorDeploymentRepo := vectordeploymentdomain.NewRepository(k8sClient)

	project := newProjectHandler(projectRepo, landscapeRepo, stageRepo, artifactDeploymentRepo, vectorDeploymentRepo, vectorPromotionRepo, vectorPromotionConfigRepo)
	api := &apiHandler{
		authHandler:    *auth,
		projectHandler: *project,
	}
	return middleware.SessionAuthentication(logger, sessionStore, authRepo, cfg, api.handler())
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
