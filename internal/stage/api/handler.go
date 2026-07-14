package api

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/konfidence-project/konfidence/internal/api/openapi"
)

func Mount(r chi.Router, _ *slog.Logger, k8s func() client.Client) {
	h := &StageHandler{k8s: k8s}
	openapi.HandlerWithOptions(openapi.NewStrictHandler(h, nil), openapi.ChiServerOptions{
		BaseURL:    "/api/v1",
		BaseRouter: r,
	})
}

type StageHandler struct{ k8s func() client.Client }

func (h *StageHandler) ListStages(_ context.Context, _ openapi.ListStagesRequestObject) (openapi.ListStagesResponseObject, error) {
	return openapi.ListStages200JSONResponse{Items: mockStages}, nil
}

func (h *StageHandler) GetStage(_ context.Context, req openapi.GetStageRequestObject) (openapi.GetStageResponseObject, error) {
	for _, s := range mockStages {
		if s.Name == req.Name {
			return openapi.GetStage200JSONResponse(s), nil
		}
	}
	return openapi.GetStage404JSONResponse{
		NotFoundJSONResponse: openapi.NotFoundJSONResponse{
			Error: struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			}{
				Code:    "not_found",
				Message: fmt.Sprintf("stage %q not found", req.Name),
			},
		},
	}, nil
}

var _ openapi.StrictServerInterface = (*StageHandler)(nil)

func mustTime(s string) time.Time { t, _ := time.Parse(time.RFC3339, s); return t }

var mockStages = []openapi.StageResponse{
	{
		Name: "prod", Namespace: "konfidence-prod", Vector: "github.com/konfidence-project/platform:1.3.9",
		Conditions: []openapi.StageCondition{
			{Type: "Ready", Status: "True", Reason: "StageReady",
				Message: "Stage is ready", LastTransitionTime: mustTime("2026-06-28T10:09:43Z")},
		},
	},
}
  package api

  import (
        "context"
        "fmt"
        "log/slog"
        "time"

        "github.com/go-chi/chi/v5"
        "sigs.k8s.io/controller-runtime/pkg/client"

        "github.com/konfidence-project/konfidence/internal/api/openapi"
  )

  func Mount(r chi.Router, _ *slog.Logger, k8s func() client.Client) {
        h := &StageHandler{k8s: k8s}
        openapi.HandlerWithOptions(openapi.NewStrictHandler(h, nil), openapi.ChiServerOptions{
                BaseURL:    "/api/v1",
                BaseRouter: r,
        })
  }

  type StageHandler struct{ k8s func() client.Client }

  func (h *StageHandler) ListStages(_ context.Context, _ openapi.ListStagesRequestObject) (openapi.ListStagesResponseObject, error) {
        return openapi.ListStages200JSONResponse{Items: mockStages}, nil
  }

  func (h *StageHandler) GetStage(_ context.Context, req openapi.GetStageRequestObject) (openapi.GetStageResponseObject, error) {
        for _, s := range mockStages {
                if s.Name == req.Name {
                        return openapi.GetStage200JSONResponse(s), nil
                }
        }
        return openapi.GetStage404JSONResponse{
                NotFoundJSONResponse: openapi.NotFoundJSONResponse{
                        Error: struct {
                                Code    string `json:"code"`
                                Message string `json:"message"`
                        }{
                                Code:    "not_found",
                                Message: fmt.Sprintf("stage %q not found", req.Name),
                        },
                },
        }, nil
  }

  var _ openapi.StrictServerInterface = (*StageHandler)(nil)
  func mustTime(s string) time.Time { t, _ := time.Parse(time.RFC3339, s); return t }
  
  var mockStages = []openapi.StageResponse{
        {Name: "prod", Namespace: "konfidence-prod", Vector: "github.com/konfidence-project/platform:1.3.9",
                Conditions: []openapi.StageCondition{{Type: "Ready", Status: "True", Reason: "StageReady", Message: "Stage is ready", LastTransitionTime: mustTime("2026-06-28T10:09:43Z")}}},
  }             
