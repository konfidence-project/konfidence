package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
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
			return getStage200Response{Items: []openapi.StageResponse{s}}, nil
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

// getStage200Response wraps a single stage in {"items": [...]} matching the StageListResponse shape
type getStage200Response struct{ Items []openapi.StageResponse }

func (r getStage200Response) VisitGetStageResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(openapi.StageListResponse{Items: r.Items})
}

func mustTime(s string) time.Time { t, _ := time.Parse(time.RFC3339, s); return t }

var mockStages = []openapi.StageResponse{
	{
		Name: "dev", Namespace: "konfidence-dev", Vector: "github.com/konfidence-project/kden:latest",
		Conditions: []openapi.StageCondition{
			{Type: "VectorDeploymentCreated", Status: "True", Reason: "DeploymentCreated",
				Message: "VectorDeployment resource created successfully", LastTransitionTime: mustTime("2026-07-01T08:12:00Z")},
			{Type: "VectorDeployed", Status: "True", Reason: "ArtifactsDeployed",
				Message: "All vector artifacts deployed and assigned", LastTransitionTime: mustTime("2026-07-01T08:15:34Z")},
			{Type: "VectorMigrated", Status: "True", Reason: "MigrationComplete",
				Message: "All migration tasks completed", LastTransitionTime: mustTime("2026-07-01T08:18:10Z")},
			{Type: "Ready", Status: "True", Reason: "StageReady",
				Message: "Stage is ready", LastTransitionTime: mustTime("2026-07-01T08:18:10Z")},
		},
	},
	{
		Name: "staging", Namespace: "konfidence-staging", Vector: "github.com/konfidence-project/konfidence:1.4.2",
		Conditions: []openapi.StageCondition{
			{Type: "VectorDeploymentCreated", Status: "True", Reason: "DeploymentCreated",
				Message: "VectorDeployment resource created successfully", LastTransitionTime: mustTime("2026-07-02T14:00:00Z")},
			{Type: "VectorDeployed", Status: "True", Reason: "ArtifactsDeployed",
				Message: "All vector artifacts deployed and assigned", LastTransitionTime: mustTime("2026-07-02T14:04:22Z")},
			{Type: "VectorMigrated", Status: "False", Reason: "MigrationFailed",
				Message: `Migration task "db-schema-v2" failed: timeout after 300s`, LastTransitionTime: mustTime("2026-07-02T14:10:05Z")},
			{Type: "Ready", Status: "False", Reason: "MigrationNotComplete",
				Message: "Stage is not ready: migration incomplete", LastTransitionTime: mustTime("2026-07-02T14:10:05Z")},
		},
	},
	{
		Name: "prod", Namespace: "konfidence-prod", Vector: "github.com/konfidence-project/platform:1.3.9",
		Conditions: []openapi.StageCondition{
			{Type: "VectorDeploymentCreated", Status: "True", Reason: "DeploymentCreated",
				Message: "VectorDeployment resource created successfully", LastTransitionTime: mustTime("2026-06-28T10:00:00Z")},
			{Type: "VectorDeployed", Status: "True", Reason: "ArtifactsDeployed",
				Message: "All vector artifacts deployed and assigned", LastTransitionTime: mustTime("2026-06-28T10:06:17Z")},
			{Type: "VectorMigrated", Status: "True", Reason: "MigrationComplete",
				Message: "All migration tasks completed", LastTransitionTime: mustTime("2026-06-28T10:09:43Z")},
			{Type: "Ready", Status: "True", Reason: "StageReady",
				Message: "Stage is ready", LastTransitionTime: mustTime("2026-06-28T10:09:43Z")},
		},
	},
	{
		Name: "canary", Namespace: "konfidence-prod", Vector: "github.com/konfidence-project/platform:1.4.2",
		Conditions: []openapi.StageCondition{
			{Type: "VectorDeploymentCreated", Status: "True", Reason: "DeploymentCreated",
				Message: "VectorDeployment resource created successfully", LastTransitionTime: mustTime("2026-07-09T07:30:00Z")},
			{Type: "VectorDeployed", Status: "Unknown", Reason: "DeploymentInProgress",
				Message: "Waiting for artifacts to be assigned", LastTransitionTime: mustTime("2026-07-09T07:30:00Z")},
		},
	},
}
