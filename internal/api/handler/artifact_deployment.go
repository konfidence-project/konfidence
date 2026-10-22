package handler

import (
	"context"
	"errors"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/api/apierror"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	artifactdeploymentdomain "github.com/konfidence-project/konfidence/internal/artifactdeployment"
	landscapedomain "github.com/konfidence-project/konfidence/internal/landscape"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (h *projectHandler) GetArtifactDeploymentV1(ctx context.Context,
	request openapi.GetArtifactDeploymentV1RequestObject) (openapi.GetArtifactDeploymentV1ResponseObject, error) {
	_, namespace, err := h.resolveProjectRequest(ctx, request.ProjectId)
	if err != nil {
		return nil, err
	}

	scope, err := h.resolveLandscapeScope(ctx, namespace, request.LandscapeId, landscapedomain.WithLandscapeId(request.LandscapeId))
	if err != nil {
		return nil, err
	}

	ad, err := h.artifactDeploymentRepo.GetForScope(ctx, scope[0], request.ArtifactDeploymentId)
	if err != nil {
		if errors.Is(err, artifactdeploymentdomain.ErrArtifactDeploymentNotFound) {
			return nil, apierror.NewNotFound("artifactDeployment", request.ArtifactDeploymentId)
		}
		return nil, apierror.NewInternal(err)
	}

	return openapi.GetArtifactDeploymentV1200JSONResponse(toArtifactDeploymentResponse(*ad)), nil
}

func (h *projectHandler) ListArtifactDeploymentsV1(ctx context.Context,
	request openapi.ListArtifactDeploymentsV1RequestObject) (openapi.ListArtifactDeploymentsV1ResponseObject, error) {
	_, namespace, err := h.resolveProjectRequest(ctx, request.ProjectId)
	if err != nil {
		return nil, err
	}

	landscapeId := ""
	var scopeOpts []landscapedomain.ScopeOption
	if request.Params.LandscapeId != nil {
		landscapeId = *request.Params.LandscapeId
		scopeOpts = append(scopeOpts, landscapedomain.WithLandscapeId(landscapeId))
	}

	scope, err := h.resolveLandscapeScope(ctx, namespace, landscapeId, scopeOpts...)
	if err != nil {
		return nil, err
	}

	var opts []artifactdeploymentdomain.ListOption
	if request.Params.VectorDeploymentId != nil {
		opts = append(opts, artifactdeploymentdomain.WithVectorDeploymentId(*request.Params.VectorDeploymentId))
	}

	ads, err := h.artifactDeploymentRepo.ListForScope(ctx, scope, opts...)
	if err != nil {
		return nil, apierror.NewInternal(err)
	}

	data := make([]openapi.ArtifactDeployment, len(ads))
	for i, resolved := range ads {
		data[i] = toArtifactDeploymentResponse(resolved)
	}

	return openapi.ListArtifactDeploymentsV1200JSONResponse{Data: data}, nil
}

func toArtifactDeploymentResponse(resolved artifactdeploymentdomain.ResolvedArtifactDeployment) openapi.ArtifactDeployment {
	ad := resolved.ArtifactDeployment

	return openapi.ArtifactDeployment{
		Id:                  ad.Name,
		LandscapeId:         resolved.LandscapeId,
		VectorDeploymentIds: resolved.VectorDeploymentIds,
		StageIds:            resolved.StageIds,
		Artifact: openapi.ArtifactReference{
			ComponentName:    ad.Spec.Component.Name,
			ComponentVersion: ad.Spec.Component.Version,
		},
		Status: openapi.ArtifactDeploymentStatus(calculateArtifactDeploymentStatus(ad.Status.Conditions)),
	}
}

func calculateArtifactDeploymentStatus(conditions []metav1.Condition) string {
	for _, cond := range conditions {
		if cond.Type == konfidence.ArtifactDeploymentReadyCondition && cond.Status == metav1.ConditionTrue {
			return "Ready"
		}
	}
	for _, cond := range conditions {
		if cond.Type == konfidence.AppHealthyCondition && cond.Status == metav1.ConditionTrue {
			return "AppHealthy"
		}
	}
	for _, cond := range conditions {
		if cond.Type == konfidence.ArtifactDeployedCondition && cond.Status == metav1.ConditionTrue {
			return "ArtifactDeployed"
		}
	}
	for _, cond := range conditions {
		if cond.Type == konfidence.ArtifactFetchedCondition && cond.Status == metav1.ConditionTrue {
			return "ArtifactFetched"
		}
	}

	// TODO: No error conditions set, should handle differently
	return konfidence.ArtifactFetchedCondition
}
