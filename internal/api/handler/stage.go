package handler

import (
	"context"
	"errors"
	"fmt"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/api/apierror"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	landscapedomain "github.com/konfidence-project/konfidence/internal/landscape"
	stagedomain "github.com/konfidence-project/konfidence/internal/stage"
)

func (h *projectHandler) GetStageV1(ctx context.Context, request openapi.GetStageV1RequestObject) (openapi.GetStageV1ResponseObject, error) {
	scope, err := h.resolveStageScope(ctx, request.ProjectId, request.LandscapeId)
	if err != nil {
		return nil, err
	}

	stage, err := h.stageRepo.GetForScope(ctx, scope, request.StageId)
	if err != nil {
		if errors.Is(err, stagedomain.ErrStageNotFound) {
			return nil, apierror.NewNotFound("stage", request.StageId)
		}
		return nil, apierror.NewInternal(err)
	}

	return openapi.GetStageV1200JSONResponse(toStageResponse(*stage)), nil
}

func (h *projectHandler) ListStagesV1(ctx context.Context, req openapi.ListStagesV1RequestObject) (openapi.ListStagesV1ResponseObject, error) {
	_, namespace, err := h.resolveProjectRequest(ctx, req.ProjectId)
	if err != nil {
		return nil, err
	}

	// A supplied landscapeId always filters, even when empty: no landscape can carry an
	// empty name, so the filter misses and the scope resolver reports it as not found.
	landscapeId := ""
	var opts []landscapedomain.ScopeOption
	if req.Params.LandscapeId != nil {
		landscapeId = *req.Params.LandscapeId
		opts = append(opts, landscapedomain.WithLandscapeId(landscapeId))
	}

	scope, err := h.resolveLandscapeScope(ctx, namespace, landscapeId, opts...)
	if err != nil {
		return nil, err
	}

	stages, err := h.stageRepo.ListForScope(ctx, scope)
	if err != nil {
		return nil, apierror.NewInternal(err)
	}

	data := make([]openapi.Stage, len(stages))
	for i, s := range stages {
		data[i] = toStageResponse(s)
	}

	return openapi.ListStagesV1200JSONResponse{Data: data}, nil
}

func (h *projectHandler) resolveStageScope(ctx context.Context, projectId string, landscapeId string) (landscapedomain.ScopedLandscape, error) {
	_, namespace, err := h.resolveProjectRequest(ctx, projectId)
	if err != nil {
		return landscapedomain.ScopedLandscape{}, err
	}

	scope, err := h.resolveLandscapeScope(ctx, namespace, landscapeId, landscapedomain.WithLandscapeId(landscapeId))
	if err != nil {
		return landscapedomain.ScopedLandscape{}, err
	}

	if len(scope) != 1 {
		return landscapedomain.ScopedLandscape{}, apierror.NewInternal(
			fmt.Errorf("resolving landscape %q returned %d entries", landscapeId, len(scope)))
	}

	return scope[0], nil
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
