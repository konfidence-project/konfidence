package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-logr/logr"
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
	"github.com/konfidence-project/konfidence/pkg/ocm/content"
	pkgcredentials "github.com/konfidence-project/konfidence/pkg/ocm/credentials"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	pkgocm "github.com/konfidence-project/konfidence/pkg/ocm/repository"
	ocmcredentials "ocm.software/open-component-model/bindings/go/credentials"
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
	logger *slog.Logger
}

var _ openapi.StrictServerInterface = (*apiHandler)(nil)

func NewAPIHandler(ctx context.Context, logger *slog.Logger, k8sClient client.Client, oidcClient oidc.Client,
	stateStore oidc.StateStore, exchangeStore oidc.ExchangeStore, sessionStore session.Store,
	cfg config.Parsed,
) (http.Handler, error) {
	auth := newAuthHandler(logger, oidcClient, stateStore, exchangeStore, sessionStore, cfg)
	authRepo := authdomain.NewRepository(k8sClient, cfg.OIDC.JWKSCacheTTL)
	projectRepo := projectdomain.NewRepository(k8sClient)
	landscapeRepo := landscapedomain.NewRepository(k8sClient)
	stageRepo := stagedomain.NewRepository(k8sClient)
	artifactDeploymentRepo := artifactdeploymentdomain.NewRepository(k8sClient)
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

	ocmReader, err := buildOCMReader(ctx, logger, k8sClient)
	if err != nil {
		return nil, fmt.Errorf("build OCM reader: %w", err)
	}

	project := newProjectHandler(projectRepo, landscapeRepo, stageRepo, artifactDeploymentRepo,
		vectorDeploymentRepo, vectorPromotionRepo, vectorPromotionConfigRepo, ocmReader)
	api := &apiHandler{
		authFlowHandler: authFlow,
		projectHandler:  *project,
		logger:          logger,
	}

	return middleware.Authenticator(logger, sessionStore, authRepo, cfg, api.handler())
}

// buildOCMReader builds a content.Reader for the API server. It reuses the
// same credential secret conventions as the vectordeployment controller, but
// reads its own env vars so the API and controller can be tuned independently.
// Verification is disabled when no signature env vars are set.
func buildOCMReader(ctx context.Context, logger *slog.Logger, k8sClient client.Client) (content.Reader, error) {
	resolver, err := buildOCMResolver(ctx, k8sClient)
	if err != nil {
		return nil, fmt.Errorf("build OCM credentials resolver: %w", err)
	}

	log := logr.FromSlogHandler(logger.Handler())
	ocmClient, err := pkgocm.NewOciClientBuilder().
		WithLogger(log).
		WithResolver(resolver).
		Build(ctx)
	if err != nil {
		return nil, fmt.Errorf("build OCI client: %w", err)
	}

	verifier, err := crypto.NewVerifierBuilder().WithLogger(log).Build()
	if err != nil {
		return nil, fmt.Errorf("build OCM verifier: %w", err)
	}

	vectorSpecs := loadSignatureSpecs("KONFIDENCE_API_VECTOR_SIGNATURES")
	artifactSpecs := loadSignatureSpecs("KONFIDENCE_API_ARTIFACT_SIGNATURES")

	return content.NewReader(ocmClient, verifier, resolver, vectorSpecs, artifactSpecs), nil
}

// buildOCMResolver loads the optional registry-credentials secret.
// Secret location is controlled by KONFIDENCE_API_CREDENTIALS_SECRET_NAME and
// KONFIDENCE_API_CREDENTIALS_SECRET_NAMESPACE. Returns (nil, nil) when unset.
func buildOCMResolver(ctx context.Context, k8sClient client.Client) (ocmcredentials.Resolver, error) {
	secretName := os.Getenv("KONFIDENCE_API_CREDENTIALS_SECRET_NAME")
	secretNs := os.Getenv("KONFIDENCE_API_CREDENTIALS_SECRET_NAMESPACE")

	switch {
	case secretName != "" && secretNs == "":
		return nil, fmt.Errorf("KONFIDENCE_API_CREDENTIALS_SECRET_NAME is set but KONFIDENCE_API_CREDENTIALS_SECRET_NAMESPACE is missing")
	case secretName == "" && secretNs != "":
		return nil, fmt.Errorf("KONFIDENCE_API_CREDENTIALS_SECRET_NAMESPACE is set but KONFIDENCE_API_CREDENTIALS_SECRET_NAME is missing")
	case secretName == "":
		return nil, nil
	}

	return pkgcredentials.ResolverFromRefs(ctx, k8sClient, secretNs, []pkgcredentials.Ref{{Name: secretName}})
}

func loadSignatureSpecs(envVar string) []crypto.SignatureSpec {
	val := strings.TrimSpace(os.Getenv(envVar))
	if val == "" {
		return nil
	}
	parts := strings.Split(val, ",")
	specs := make([]crypto.SignatureSpec, 0, len(parts))
	for _, p := range parts {
		if n := strings.TrimSpace(p); n != "" {
			specs = append(specs, crypto.DefaultSignatureSpec(n, nil))
		}
	}
	return specs
}

func (s *apiHandler) handler() http.Handler {
	errHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		if apiErr := apierror.As(err); apiErr != nil {
			if apiErr.Err != nil {
				s.logger.Error("request failed",
					"method", r.Method,
					"path", r.URL.Path,
					"code", apiErr.Code,
					"error", apiErr.Err.Error(),
				)
			}
			apierror.Write(w, apiErr)
			return
		}
		s.logger.Error("unhandled error",
			"method", r.Method,
			"path", r.URL.Path,
			"error", err.Error(),
		)
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
