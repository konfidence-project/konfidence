package handler // nolint

import (
	"context"

	"github.com/konfidence-project/konfidence/internal/api/openapi"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InfoHandler struct{ k8s func() client.Client }

func (h *InfoHandler) GetHealthStatus(_ context.Context, _ openapi.GetHealthStatusRequestObject) (openapi.GetHealthStatusResponseObject, error) {
	return openapi.GetHealthStatus200JSONResponse{Status: "ok"}, nil
}

func (h *InfoHandler) GetReadinessStatus(_ context.Context, _ openapi.GetReadinessStatusRequestObject) (openapi.GetReadinessStatusResponseObject, error) {
	return openapi.GetReadinessStatus200JSONResponse{Status: "ok"}, nil
}
