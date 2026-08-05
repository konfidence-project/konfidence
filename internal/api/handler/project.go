package handler // nolint

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
)

type projectHandler struct {
	projectRepo   projectdomain.Repository
	landscapeRepo landscapedomain.Repository
}

func newProjectHandler(projectRepo projectdomain.Repository, landscapeRepo landscapedomain.Repository) *projectHandler {
	return &projectHandler{
		projectRepo:   projectRepo,
		landscapeRepo: landscapeRepo,
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

func toProjectResponse(p konfidence.Project) openapi.Project {
	return openapi.Project{
		Id:   p.Name,
		Name: p.Spec.DisplayName,
	}
}

func (h *projectHandler) ListLandscapesV1(ctx context.Context, req openapi.ListLandscapesV1RequestObject) (openapi.ListLandscapesV1ResponseObject, error) {
	identity, err := session.FromContext(ctx)
	if err != nil {
		return openapi.ListLandscapesV1401JSONResponse{
			UnauthorizedJSONResponse: apierror.NewUnauthorizedResponse(),
		}, nil
	}

	project, err := h.projectRepo.Get(ctx, req.ProjectId)
	if err != nil {
		if errors.Is(err, projectdomain.ErrNotFound) {
			return openapi.ListLandscapesV1403JSONResponse{
				ForbiddenJSONResponse: apierror.NewForbiddenResponse(fmt.Sprintf("access to project %q is not allowed", req.ProjectId)),
			}, nil
		}
		return openapi.ListLandscapesV1500JSONResponse{
			InternalErrorJSONResponse: apierror.NewInternalErrorResponse(),
		}, nil
	}

	if !callerHasProjectAccess(identity, project) {
		return openapi.ListLandscapesV1403JSONResponse{
			ForbiddenJSONResponse: apierror.NewForbiddenResponse(fmt.Sprintf("access to project %q is not allowed", req.ProjectId)),
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

func callerHasProjectAccess(identity *session.Context, project *konfidence.Project) bool {
	if len(project.Spec.RoleBindings) == 0 {
		return true
	}
	_, hasAccess := identity.ProjectRoles[project.Name]
	return hasAccess
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