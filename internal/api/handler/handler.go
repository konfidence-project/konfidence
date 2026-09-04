package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

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
	stagedomain "github.com/konfidence-project/konfidence/internal/stage"
	vectordeploymentdomain "github.com/konfidence-project/konfidence/internal/vectordeployment"
	vectorpromotiondomain "github.com/konfidence-project/konfidence/internal/vectorpromotion"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type authFlowHandler interface {
	LoginV1(ctx context.Context, request openapi.LoginV1RequestObject) (openapi.LoginV1ResponseObject, error)
	AuthCallbackV1(ctx context.Context, request openapi.AuthCallbackV1RequestObject) (openapi.AuthCallbackV1ResponseObject, error)
	LogoutV1(ctx context.Context, request openapi.LogoutV1RequestObject) (openapi.LogoutV1ResponseObject, error)
	GetIdentityV1(ctx context.Context, request openapi.GetIdentityV1RequestObject) (openapi.GetIdentityV1ResponseObject, error)
	PostExchangeCodeV1(ctx context.Context, request openapi.PostExchangeCodeV1RequestObject) (openapi.PostExchangeCodeV1ResponseObject, error)
}

type apiHandler struct {
	authFlowHandler
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
	vectorPromotionRepo := vectorpromotiondomain.NewRepository(k8sClient)
	vectorPromotionConfigRepo := vectorpromotiondomain.NewConfigRepository(k8sClient)
	vectorDeploymentRepo := vectordeploymentdomain.NewRepository(k8sClient)

	var authFlow authFlowHandler
	if cfg.OIDC.Enabled {
		authFlow = auth
	} else {
		authFlow = &fakeAuthHandler{
			auth:          auth,
			exchangeStore: exchangeStore,
			serverBaseURL: serverBaseURL(cfg.Server.Addr),
		}
	}

	project := newProjectHandler(projectRepo, landscapeRepo, stageRepo, vectorDeploymentRepo, vectorPromotionRepo, vectorPromotionConfigRepo)
	api := &apiHandler{
		authFlowHandler: authFlow,
		projectHandler:  *project,
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

// serverBaseURL converts a listen address (e.g. ":8090" or "0.0.0.0:8090") to
// an absolute localhost URL used to build redirect targets in no-auth mode.
func serverBaseURL(addr string) string {
	if !strings.Contains(addr, ":") {
		return "http://localhost"
	}
	port := addr[strings.LastIndex(addr, ":"):]
	return "http://localhost" + port
}
