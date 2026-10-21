package handler

import (
	"context"
	"errors"
	"fmt"

	"github.com/konfidence-project/konfidence/internal/api/apierror"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	landscapedomain "github.com/konfidence-project/konfidence/internal/landscape"
	vectordeploymentdomain "github.com/konfidence-project/konfidence/internal/vectordeployment"
	"github.com/konfidence-project/konfidence/pkg/ocm/compref"
)

func (h *projectHandler) GetVectorDeploymentV1(ctx context.Context,
	request openapi.GetVectorDeploymentV1RequestObject) (openapi.GetVectorDeploymentV1ResponseObject, error) {
	_, namespace, err := h.resolveProjectRequest(ctx, request.ProjectId)
	if err != nil {
		return nil, err
	}

	scope, err := h.resolveLandscapeScope(ctx, namespace, request.LandscapeId, landscapedomain.WithLandscapeId(request.LandscapeId))
	if err != nil {
		return nil, err
	}

	deployment, err := h.vectorDeploymentRepo.GetForScope(ctx, scope[0], request.VectorDeploymentId)
	if err != nil {
		if errors.Is(err, vectordeploymentdomain.ErrVectorDeploymentNotFound) {
			return nil, apierror.NewNotFound("vectorDeployment", request.VectorDeploymentId)
		}
		return nil, apierror.NewInternal(err)
	}

	response, err := toVectorDeploymentResponse(*deployment)
	if err != nil {
		return nil, apierror.NewInternal(err)
	}
	return openapi.GetVectorDeploymentV1200JSONResponse(response), nil
}

func (h *projectHandler) ListVectorDeploymentsV1(ctx context.Context,
	req openapi.ListVectorDeploymentsV1RequestObject) (openapi.ListVectorDeploymentsV1ResponseObject, error) {
	_, namespace, err := h.resolveProjectRequest(ctx, req.ProjectId)
	if err != nil {
		return nil, err
	}

	landscapeId := ""
	var opts []landscapedomain.ScopeOption
	if req.Params.LandscapeId != nil {
		landscapeId = *req.Params.LandscapeId
		opts = append(opts, landscapedomain.WithLandscapeId(landscapeId))
	}

	var scope []landscapedomain.ScopedLandscape
	scope, err = h.resolveLandscapeScope(ctx, namespace, landscapeId, opts...)
	if err != nil {
		return nil, err
	}

	deployments, err := h.vectorDeploymentRepo.ListForScope(ctx, scope)
	if err != nil {
		return nil, apierror.NewInternal(err)
	}

	data := make([]openapi.VectorDeployment, len(deployments))
	for i, deployment := range deployments {
		data[i], err = toVectorDeploymentResponse(deployment)
		if err != nil {
			return nil, apierror.NewInternal(err)
		}
	}
	return openapi.ListVectorDeploymentsV1200JSONResponse{Data: data}, nil
}

func toVectorDeploymentResponse(resolvedVectorDeployment vectordeploymentdomain.ResolvedVectorDeployment) (openapi.VectorDeployment, error) {
	ref, err := compref.ParseComponentVersionReference(resolvedVectorDeployment.VectorDeployment.Spec.Vector)
	if err != nil {
		return openapi.VectorDeployment{}, fmt.Errorf("parsing vector reference: %w", err)
	}
	return openapi.VectorDeployment{
		Id:          resolvedVectorDeployment.VectorDeployment.Name,
		LandscapeId: resolvedVectorDeployment.LandscapeId,
		StageId:     resolvedVectorDeployment.StageId,
		Vector: openapi.VectorReference{
			Repository:       ref.Repository,
			ComponentName:    ref.Component,
			ComponentVersion: ref.Version,
		},
		Status: openapi.VectorDeploymentStatus(vectordeploymentdomain.StateFromConditions(resolvedVectorDeployment.VectorDeployment.Status.Conditions)),
	}, nil
}
