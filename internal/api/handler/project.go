package handler // nolint

import (
	"context"

	"github.com/konfidence-project/konfidence/internal/api/openapi"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ProjectHandler struct{ k8s client.Client }

func NewProjectHandler(k8s client.Client) *ProjectHandler {
	return &ProjectHandler{k8s: k8s}
}

func (h *ProjectHandler) ListProjects(_ context.Context, _ openapi.ListProjectsRequestObject) (openapi.ListProjectsResponseObject, error) {
	return nil, nil
}

func (h *ProjectHandler) ListLandscapes(_ context.Context, _ openapi.ListLandscapesRequestObject) (openapi.ListLandscapesResponseObject, error) {
	return nil, nil
}

func (h *ProjectHandler) ListStages(_ context.Context, _ openapi.ListStagesRequestObject) (openapi.ListStagesResponseObject, error) {
	return nil, nil
}

func (h *ProjectHandler) ListVectorDeployments(_ context.Context,
	_ openapi.ListVectorDeploymentsRequestObject) (openapi.ListVectorDeploymentsResponseObject, error) {
	return nil, nil
}

func (h *ProjectHandler) ListArtifactDeployments(_ context.Context,
	_ openapi.ListArtifactDeploymentsRequestObject) (openapi.ListArtifactDeploymentsResponseObject, error) {
	return nil, nil
}
