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

func (h *ProjectHandler) ListProjectsV1(_ context.Context, _ openapi.ListProjectsV1RequestObject) (openapi.ListProjectsV1ResponseObject, error) {
	return openapi.ListProjectsV1200JSONResponse{
		Data: []openapi.Project{
			{Id: "sample-project", Name: "Sample Project"},
		},
	}, nil
}

func (h *ProjectHandler) ListLandscapesV1(_ context.Context, _ openapi.ListLandscapesV1RequestObject) (openapi.ListLandscapesV1ResponseObject, error) {
	return nil, nil
}

func (h *ProjectHandler) ListStagesV1(_ context.Context, _ openapi.ListStagesV1RequestObject) (openapi.ListStagesV1ResponseObject, error) {
	return nil, nil
}

func (h *ProjectHandler) ListVectorDeploymentsV1(_ context.Context,
	_ openapi.ListVectorDeploymentsV1RequestObject) (openapi.ListVectorDeploymentsV1ResponseObject, error) {
	return nil, nil
}

func (h *ProjectHandler) ListArtifactDeploymentsV1(_ context.Context,
	_ openapi.ListArtifactDeploymentsV1RequestObject) (openapi.ListArtifactDeploymentsV1ResponseObject, error) {
	return nil, nil
}
