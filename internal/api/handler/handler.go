package handler

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Mount(r chi.Router, _ *slog.Logger, k8s func() (client.Client, error)) {
	h := NewServerHandler(k8s)
	errHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		if apiErr := AsAPIError(err); apiErr != nil {
			WriteAPIError(w, apiErr)
			return
		}
		WriteInternalError(w)
	}
	openapi.HandlerWithOptions(openapi.NewStrictHandlerWithOptions(h, nil, openapi.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  errHandler,
		ResponseErrorHandlerFunc: errHandler,
	}), openapi.ChiServerOptions{
		BaseURL:    "/api/v1",
		BaseRouter: r,
	})
}

func NewServerHandler(k8s func() (client.Client, error)) *ServerHandler {
	return &ServerHandler{
		InfoHandler{k8s: k8s},
		AuthHandler{k8s: k8s},
		ProjectHandler{k8s: k8s},
	}
}

type ServerHandler struct {
	InfoHandler
	AuthHandler
	ProjectHandler
}

var _ openapi.StrictServerInterface = (*ServerHandler)(nil)
