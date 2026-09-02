package handler

import (
	"context"
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
	vectorpromotiondomain "github.com/konfidence-project/konfidence/internal/vectorpromotion"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type authFlowHandler interface {
	LoginV1(ctx context.Context, request openapi.LoginV1RequestObject) (openapi.LoginV1ResponseObject, error)
	AuthCallbackV1(ctx context.Context, request openapi.AuthCallbackV1RequestObject) (openapi.AuthCallbackV1ResponseObject, error)
	LogoutV1(ctx context.Context, request openapi.LogoutV1RequestObject) (openapi.LogoutV1ResponseObject, error)
}

type apiHandler struct {
	authFlowHandler
	auth *authHandler
	projectHandler
}

var _ openapi.StrictServerInterface = (*apiHandler)(nil)

func (a *apiHandler) GetIdentityV1(ctx context.Context, req openapi.GetIdentityV1RequestObject) (openapi.GetIdentityV1ResponseObject, error) {
	return a.auth.GetIdentityV1(ctx, req)
}

func (a *apiHandler) PostExchangeCodeV1(ctx context.Context, req openapi.PostExchangeCodeV1RequestObject) (openapi.PostExchangeCodeV1ResponseObject, error) {
	return a.auth.PostExchangeCodeV1(ctx, req)
}

func NewAPIHandler(logger *slog.Logger, k8sClient client.Client, oidcClient oidc.Client,
	stateStore oidc.StateStore, exchangeStore oidc.ExchangeStore, sessionStore session.Store,
	cfg config.Parsed,
) (http.Handler, error) {
	auth := newAuthHandler(logger, oidcClient, stateStore, exchangeStore, sessionStore, cfg)
	authRepo := authdomain.NewRepository(k8sClient)
	projectRepo := projectdomain.NewRepository(k8sClient)
	landscapeRepo := landscapedomain.NewRepository(k8sClient)
	vectorPromotionRepo := vectorpromotiondomain.NewRepository(k8sClient)
	vectorPromotionConfigRepo := vectorpromotiondomain.NewConfigRepository(k8sClient)

	var authFlow authFlowHandler
	if cfg.OIDC.Enabled {
		authFlow = auth
	} else {
		authFlow = &fakeAuthHandler{sessions: sessionStore, config: cfg}
	}

	project := newProjectHandler(projectRepo, landscapeRepo, vectorPromotionRepo, vectorPromotionConfigRepo)
	api := &apiHandler{
		authFlowHandler: authFlow,
		auth:            auth,
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
