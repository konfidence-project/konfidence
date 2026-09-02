package handler

import (
	"context"
	"errors"
	"fmt"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/api/apierror"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	landscapedomain "github.com/konfidence-project/konfidence/internal/landscape"
	projectdomain "github.com/konfidence-project/konfidence/internal/project"
	stagedomain "github.com/konfidence-project/konfidence/internal/stage"
)

func (h *projectHandler) ListStagesV1(ctx context.Context, req openapi.ListStagesV1RequestObject) (openapi.ListStagesV1ResponseObject, error) {
	project, err := h.resolveProject(ctx, req.ProjectId)
	if err != nil {
		switch {
		case errors.Is(err, errNoSession):
			return openapi.ListStagesV1401JSONResponse{
				UnauthorizedJSONResponse: apierror.NewUnauthorizedResponse(),
			}, nil
		case errors.Is(err, projectdomain.ErrForbidden):
			return openapi.ListStagesV1403JSONResponse{
				ForbiddenJSONResponse: apierror.NewForbiddenResponse(fmt.Sprintf("access to project %q is not allowed", req.ProjectId)),
			}, nil
		case errors.Is(err, projectdomain.ErrNotFound):
			return openapi.ListStagesV1404JSONResponse{
				NotFoundJSONResponse: apierror.NewNotFoundResponse(fmt.Sprintf("project %q not found", req.ProjectId)),
			}, nil
		default:
			return openapi.ListStagesV1500JSONResponse{
				InternalErrorJSONResponse: apierror.NewInternalErrorResponse(),
			}, nil
		}
	}

	// A supplied landscapeId always filters, even when empty: no landscape can carry an
	// empty name, so the filter misses and the scope resolver reports it as not found.
	landscapeId := ""
	var opts []landscapedomain.ScopeOption
	if req.Params.LandscapeId != nil {
		landscapeId = *req.Params.LandscapeId
		opts = append(opts, landscapedomain.WithLandscapeId(landscapeId))
	}

	scope, err := h.landscapeRepo.ResolveScope(ctx, project.Status.Namespace, opts...)
	if err != nil {
		if errors.Is(err, landscapedomain.ErrLandscapeNotFound) {
			return openapi.ListStagesV1404JSONResponse{
				NotFoundJSONResponse: apierror.NewNotFoundResponse(fmt.Sprintf("landscape %q not found", landscapeId)),
			}, nil
		}
		return openapi.ListStagesV1500JSONResponse{
			InternalErrorJSONResponse: apierror.NewInternalErrorResponse(),
		}, nil
	}

	stages, err := h.stageRepo.ListForScope(ctx, scope)
	if err != nil {
		return openapi.ListStagesV1500JSONResponse{
			InternalErrorJSONResponse: apierror.NewInternalErrorResponse(),
		}, nil
	}

	data := make([]openapi.Stage, len(stages))
	for i, s := range stages {
		data[i] = toStageResponse(s)
	}

	return openapi.ListStagesV1200JSONResponse{Data: data}, nil
}

// toStageResponse maps a resolved stage to its DTO. The Stage CRD has no display name,
// so the stage name doubles as its id.
func toStageResponse(rs stagedomain.ResolvedStage) openapi.Stage {
	return openapi.Stage{
		Id:                 rs.Stage.Name,
		Name:               rs.Stage.Name,
		LandscapeId:        rs.LandscapeID,
		TargetStageVersion: toStageVersionResponse(rs.Target),
		ActiveStageVersion: toStageVersionResponse(rs.Active),
	}
}

// toStageVersionResponse maps a stage version to its DTO. A nil version stays nil so the
// field is absent in the response.
func toStageVersionResponse(v *konfidence.StageVersion) *openapi.StageVersion {
	if v == nil {
		return nil
	}
	return &openapi.StageVersion{
		Id:              v.Name,
		Vector:          v.Spec.Vector,
		StageGeneration: int(v.Spec.StageGeneration),
		Status:          openapi.StageVersionStatus(stagedomain.StateFromConditions(v.Status.Conditions)),
	}
}
