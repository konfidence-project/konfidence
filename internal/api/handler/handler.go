package handler

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Mount(r chi.Router, _ *slog.Logger, k8s func() client.Client) {
	h := NewServerHandler(k8s)
	openapi.HandlerWithOptions(openapi.NewStrictHandler(h, nil), openapi.ChiServerOptions{
		BaseURL:    "/api/v1",
		BaseRouter: r,
	})
}

func NewServerHandler(k8s func() client.Client) *ServerHandler {
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
