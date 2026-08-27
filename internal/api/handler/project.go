package handler

import (
	"context"
	"errors"
	"fmt"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/api/apierror"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	"github.com/konfidence-project/konfidence/internal/api/session"
	landscapedomain "github.com/konfidence-project/konfidence/internal/landscape"
	projectdomain "github.com/konfidence-project/konfidence/internal/project"
	vectorpromotiondomain "github.com/konfidence-project/konfidence/internal/vectorpromotion"
)

type projectHandler struct {
	projectRepo               projectdomain.Repository
	landscapeRepo             landscapedomain.Repository
	vectorPromotionRepo       vectorpromotiondomain.Repository
	vectorPromotionConfigRepo vectorpromotiondomain.ConfigRepository
}

func newProjectHandler(projectRepo projectdomain.Repository, landscapeRepo landscapedomain.Repository,
	vectorPromotionRepo vectorpromotiondomain.Repository, vectorPromotionConfigRepo vectorpromotiondomain.ConfigRepository) *projectHandler {
	return &projectHandler{
		projectRepo:               projectRepo,
		landscapeRepo:             landscapeRepo,
		vectorPromotionRepo:       vectorPromotionRepo,
		vectorPromotionConfigRepo: vectorPromotionConfigRepo,
	}
}

func (h *projectHandler) ListProjectsV1(ctx context.Context, _ openapi.ListProjectsV1RequestObject) (openapi.ListProjectsV1ResponseObject, error) {
	identity, err := session.FromContext(ctx)
	if err != nil {
		return openapi.ListProjectsV1401JSONResponse{
			UnauthorizedJSONResponse: apierror.NewUnauthorizedResponse(),
		}, nil
	}
	projects, err := h.projectRepo.List(ctx, identity.ProjectRoles)
	if err != nil {
		return openapi.ListProjectsV1500JSONResponse{
			InternalErrorJSONResponse: apierror.NewInternalErrorResponse(),
		}, nil
	}

	data := make([]openapi.Project, 0, len(projects))
	for _, p := range projects {
		data = append(data, toProjectResponse(p))
	}

	return openapi.ListProjectsV1200JSONResponse{Data: data}, nil
}

func (h *projectHandler) ListLandscapesV1(ctx context.Context, req openapi.ListLandscapesV1RequestObject) (openapi.ListLandscapesV1ResponseObject, error) {
	identity, err := session.FromContext(ctx)
	if err != nil {
		return openapi.ListLandscapesV1401JSONResponse{
			UnauthorizedJSONResponse: apierror.NewUnauthorizedResponse(),
		}, nil
	}

	project, err := h.projectRepo.Get(ctx, req.ProjectId, identity.ProjectRoles)
	if err != nil {
		if errors.Is(err, projectdomain.ErrNotFound) {
			return openapi.ListLandscapesV1404JSONResponse{
				NotFoundJSONResponse: apierror.NewNotFoundResponse(fmt.Sprintf("project %q not found", req.ProjectId)),
			}, nil
		}
		if errors.Is(err, projectdomain.ErrForbidden) {
			return openapi.ListLandscapesV1403JSONResponse{
				ForbiddenJSONResponse: apierror.NewForbiddenResponse(fmt.Sprintf("access to project %q is not allowed", req.ProjectId)),
			}, nil
		}
		return openapi.ListLandscapesV1500JSONResponse{
			InternalErrorJSONResponse: apierror.NewInternalErrorResponse(),
		}, nil
	}

	if project.Status.Namespace == "" {
		return openapi.ListLandscapesV1500JSONResponse{
			InternalErrorJSONResponse: apierror.NewInternalErrorResponse(),
		}, nil
	}

	landscapes, err := h.landscapeRepo.ListForProject(ctx, project.Status.Namespace)
	if err != nil {
		return openapi.ListLandscapesV1500JSONResponse{
			InternalErrorJSONResponse: apierror.NewInternalErrorResponse(),
		}, nil
	}

	data := make([]openapi.Landscape, len(landscapes))
	for i, l := range landscapes {
		data[i] = toLandscapeResponse(l)
	}

	return openapi.ListLandscapesV1200JSONResponse{Data: data}, nil
}

func toProjectResponse(p konfidence.Project) openapi.Project {
	return openapi.Project{
		Id:   p.Name,
		Name: p.Spec.DisplayName,
	}
}

func toLandscapeResponse(l konfidence.Landscape) openapi.Landscape {
	return openapi.Landscape{
		Id:   l.Name,
		Name: l.Spec.DisplayName,
	}
}

func (h *projectHandler) ListStagesV1(_ context.Context, _ openapi.ListStagesV1RequestObject) (openapi.ListStagesV1ResponseObject, error) {
	return nil, nil
}

func (h *projectHandler) ListVectorDeploymentsV1(_ context.Context,
	_ openapi.ListVectorDeploymentsV1RequestObject) (openapi.ListVectorDeploymentsV1ResponseObject, error) {
	return nil, nil
}

func (h *projectHandler) ListArtifactDeploymentsV1(_ context.Context,
	_ openapi.ListArtifactDeploymentsV1RequestObject) (openapi.ListArtifactDeploymentsV1ResponseObject, error) {
	return nil, nil
}

func (h *projectHandler) GetVectorPromotionConfigV1(ctx context.Context,
	request openapi.GetVectorPromotionConfigV1RequestObject) (openapi.GetVectorPromotionConfigV1ResponseObject, error) {
	identity, err := session.FromContext(ctx)
	if err != nil {
		return openapi.GetVectorPromotionConfigV1401JSONResponse{
			UnauthorizedJSONResponse: apierror.NewUnauthorizedResponse(),
		}, nil
	}

	if !identity.IsAuthenticatedForProject(request.ProjectId) {
		return openapi.GetVectorPromotionConfigV1403JSONResponse{
			ForbiddenJSONResponse: apierror.NewForbiddenResponse(""),
		}, nil
	}

	config, err := h.vectorPromotionConfigRepo.Get(ctx, request.ProjectId, request.VectorPromotionConfigId)
	if err != nil {
		return openapi.GetVectorPromotionConfigV1500JSONResponse{
			InternalErrorJSONResponse: apierror.NewInternalErrorResponse(),
		}, nil
	}

	if config == nil {
		return openapi.GetVectorPromotionConfigV1404JSONResponse{
			NotFoundJSONResponse: apierror.NewNotFoundResponse(""),
		}, nil
	}

	response := toVectorPromotionConfigResponse(*config)
	return openapi.GetVectorPromotionConfigV1200JSONResponse(response), nil
}

func (h *projectHandler) ListVectorPromotionConfigsV1(ctx context.Context,
	request openapi.ListVectorPromotionConfigsV1RequestObject) (openapi.ListVectorPromotionConfigsV1ResponseObject, error) {
	identity, err := session.FromContext(ctx)
	if err != nil {
		return openapi.ListVectorPromotionConfigsV1401JSONResponse{
			UnauthorizedJSONResponse: apierror.NewUnauthorizedResponse(),
		}, nil
	}

	if !identity.IsAuthenticatedForProject(request.ProjectId) {
		return openapi.ListVectorPromotionConfigsV1403JSONResponse{
			ForbiddenJSONResponse: apierror.NewForbiddenResponse(""),
		}, nil
	}

	configs, err := h.vectorPromotionConfigRepo.List(ctx)
	if err != nil {
		return openapi.ListVectorPromotionConfigsV1500JSONResponse{
			InternalErrorJSONResponse: apierror.NewInternalErrorResponse(),
		}, nil
	}

	data := make([]openapi.VectorPromotionConfig, 0, len(configs))
	for _, c := range configs {
		data = append(data, toVectorPromotionConfigResponse(c))
	}

	return openapi.ListVectorPromotionConfigsV1200JSONResponse{Data: data}, nil
}

func toVectorPromotionConfigResponse(_ konfidence.VectorPromotionConfig) openapi.VectorPromotionConfig {
	// TODO implement mapping
	return openapi.VectorPromotionConfig{}
}

func (h *projectHandler) GetVectorPromotionV1(ctx context.Context,
	request openapi.GetVectorPromotionV1RequestObject) (openapi.GetVectorPromotionV1ResponseObject, error) {
	identity, err := session.FromContext(ctx)
	if err != nil {
		return openapi.GetVectorPromotionV1401JSONResponse{
			UnauthorizedJSONResponse: apierror.NewUnauthorizedResponse(),
		}, nil
	}

	if !identity.IsAuthenticatedForProject(request.ProjectId) {
		return openapi.GetVectorPromotionV1403JSONResponse{
			ForbiddenJSONResponse: apierror.NewForbiddenResponse(""),
		}, nil
	}

	promotion, err := h.vectorPromotionRepo.Get(ctx, request.ProjectId, request.VectorPromotionId)
	if err != nil {
		return openapi.GetVectorPromotionV1500JSONResponse{
			InternalErrorJSONResponse: apierror.NewInternalErrorResponse(),
		}, nil
	}

	if promotion == nil {
		return openapi.GetVectorPromotionV1404JSONResponse{
			NotFoundJSONResponse: apierror.NewNotFoundResponse(""),
		}, nil
	}

	response := toVectorPromotionResponse(*promotion)
	return openapi.GetVectorPromotionV1200JSONResponse(response), nil
}

func (h *projectHandler) ApproveVectorPromotionV1(ctx context.Context,
	request openapi.ApproveVectorPromotionV1RequestObject) (openapi.ApproveVectorPromotionV1ResponseObject, error) {
	identity, err := session.FromContext(ctx)
	if err != nil {
		return openapi.ApproveVectorPromotionV1401JSONResponse{
			UnauthorizedJSONResponse: apierror.NewUnauthorizedResponse(),
		}, nil
	}

	if !identity.IsAuthenticatedForProject(request.ProjectId) {
		return openapi.ApproveVectorPromotionV1403JSONResponse{
			ForbiddenJSONResponse: apierror.NewForbiddenResponse(""),
		}, nil
	}

	err = h.vectorPromotionRepo.Approve(ctx, request.ProjectId, request.VectorPromotionId)
	if err != nil {
		return openapi.ApproveVectorPromotionV1500JSONResponse{
			InternalErrorJSONResponse: apierror.NewInternalErrorResponse(),
		}, nil
	}

	return openapi.ApproveVectorPromotionV1204Response{}, nil
}

func toVectorPromotionResponse(_ konfidence.VectorPromotion) openapi.VectorPromotion {
	// TODO implement mapping
	return openapi.VectorPromotion{}
}
