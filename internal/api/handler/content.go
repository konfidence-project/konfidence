package handler

import (
	"context"
	"errors"
	"fmt"

	"github.com/konfidence-project/konfidence/internal/api/apierror"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	"github.com/konfidence-project/konfidence/internal/api/session"
	landscapedomain "github.com/konfidence-project/konfidence/internal/landscape"
	stagedomain "github.com/konfidence-project/konfidence/internal/stage"
	vectorpromotiondomain "github.com/konfidence-project/konfidence/internal/vectorpromotion"
	kompref "github.com/konfidence-project/konfidence/pkg/ocm/compref"
	pkgcontent "github.com/konfidence-project/konfidence/pkg/ocm/content"
	pkgocm "github.com/konfidence-project/konfidence/pkg/ocm/repository"
)

func (h *projectHandler) GetVectorContentV1(ctx context.Context, request openapi.GetVectorContentV1RequestObject) (openapi.GetVectorContentV1ResponseObject, error) {
	identity, err := session.FromContext(ctx)
	if err != nil {
		return nil, apierror.NewUnauthorized()
	}

	if _, err := h.resolveProjectNamespace(ctx, identity, request.ProjectId); err != nil {
		return nil, err
	}

	ref, err := kompref.ParseComponentVersionReference(request.Params.Ref)
	if err != nil {
		return nil, apierror.NewInternal(err)
	}

	vectorContent, err := h.ocmReader.GetVectorContent(ctx, ref)
	if err != nil {
		if errors.Is(err, pkgocm.ErrNotFound) {
			return nil, apierror.NewNotFound("vector", request.Params.Ref)
		}
		return nil, apierror.NewInternal(err)
	}

	return openapi.GetVectorContentV1200JSONResponse(openapi.VectorContent{
		Ref:        request.Params.Ref,
		Components: toVectorComponents(vectorContent.Components),
		Config:     toVectorConfig(vectorContent.Config),
	}), nil
}

func (h *projectHandler) GetArtifactContentV1(ctx context.Context, request openapi.GetArtifactContentV1RequestObject) (openapi.GetArtifactContentV1ResponseObject, error) {
	identity, err := session.FromContext(ctx)
	if err != nil {
		return nil, apierror.NewUnauthorized()
	}

	if _, err := h.resolveProjectNamespace(ctx, identity, request.ProjectId); err != nil {
		return nil, err
	}

	ref, err := kompref.ParseComponentVersionReference(request.Params.Ref)
	if err != nil {
		return nil, apierror.NewInternal(err)
	}

	artifactContent, err := h.ocmReader.GetArtifactContent(ctx, ref)
	if err != nil {
		if errors.Is(err, pkgocm.ErrNotFound) {
			return nil, apierror.NewNotFound("artifact", request.Params.Ref)
		}
		return nil, apierror.NewInternal(err)
	}

	result := openapi.ArtifactContent{
		Ref:        request.Params.Ref,
		Type:       artifactContent.Type,
		AllowReuse: artifactContent.AllowReuse,
	}
	if len(artifactContent.Tasks) > 0 {
		tasks := toArtifactTasks(artifactContent.Tasks)
		result.Tasks = &tasks
	}
	if len(artifactContent.Resources) > 0 {
		resources := toArtifactResources(artifactContent.Resources)
		result.Resources = &resources
	}
	return openapi.GetArtifactContentV1200JSONResponse(result), nil
}

func toArtifactTasks(ts []pkgcontent.ArtifactTask) []openapi.ArtifactTask {
	out := make([]openapi.ArtifactTask, len(ts))
	for i, t := range ts {
		entry := openapi.ArtifactTask{
			Name: t.Name,
			Type: t.Type,
		}
		if len(t.DependsOn) > 0 {
			d := make([]string, len(t.DependsOn))
			copy(d, t.DependsOn)
			entry.DependsOn = &d
		}
		out[i] = entry
	}
	return out
}

func toArtifactResources(rs []pkgcontent.ArtifactResource) []openapi.ArtifactResource {
	out := make([]openapi.ArtifactResource, len(rs))
	for i, r := range rs {
		out[i] = openapi.ArtifactResource{
			Name: r.Name,
			Type: r.Type,
		}
	}
	return out
}

func (h *projectHandler) GetVectorPromotionDiffV1(ctx context.Context, request openapi.GetVectorPromotionDiffV1RequestObject) (openapi.GetVectorPromotionDiffV1ResponseObject, error) {
	identity, err := session.FromContext(ctx)
	if err != nil {
		return nil, apierror.NewUnauthorized()
	}

	namespace, err := h.resolveProjectNamespace(ctx, identity, request.ProjectId)
	if err != nil {
		return nil, err
	}

	promotion, err := h.vectorPromotionRepo.Get(ctx, namespace, request.VectorPromotionId)
	if err != nil {
		if errors.Is(err, vectorpromotiondomain.ErrVectorPromotionNotFound) {
			return nil, apierror.NewNotFound("vectorPromotion", request.VectorPromotionId)
		}
		return nil, apierror.NewInternal(err)
	}

	incomingRef, err := kompref.ParseComponentVersionReference(promotion.Spec.Vector)
	if err != nil {
		return nil, apierror.NewInternal(fmt.Errorf("invalid incoming vector ref %q: %w", promotion.Spec.Vector, err))
	}

	scope, err := h.resolveLandscapeScope(ctx, namespace, promotion.Spec.Target.Landscape,
		landscapedomain.WithLandscapeId(promotion.Spec.Target.Landscape))
	if err != nil {
		return nil, err
	}

	stage, err := h.stageRepo.GetForScope(ctx, scope[0], promotion.Spec.Target.Name)
	if err != nil {
		if errors.Is(err, stagedomain.ErrNotFound) {
			return nil, apierror.NewNotFound("stage", promotion.Spec.Target.Name)
		}
		return nil, apierror.NewInternal(err)
	}

	if stage.Stage.Spec.Vector == "" {
		return nil, apierror.NewNotFound("current vector", promotion.Spec.Target.Name)
	}

	currentRef, err := kompref.ParseComponentVersionReference(stage.Stage.Spec.Vector)
	if err != nil {
		return nil, apierror.NewInternal(fmt.Errorf("invalid current vector ref %q: %w", stage.Stage.Spec.Vector, err))
	}

	incomingVectorContent, err := h.ocmReader.GetVectorContent(ctx, incomingRef)
	if err != nil {
		if errors.Is(err, pkgocm.ErrNotFound) {
			return nil, apierror.NewNotFound("incoming vector", promotion.Spec.Vector)
		}
		return nil, apierror.NewInternal(err)
	}

	currentVectorContent, err := h.ocmReader.GetVectorContent(ctx, currentRef)
	if err != nil {
		if errors.Is(err, pkgocm.ErrNotFound) {
			return nil, apierror.NewNotFound("current vector", stage.Stage.Spec.Vector)
		}
		return nil, apierror.NewInternal(err)
	}

	return openapi.GetVectorPromotionDiffV1200JSONResponse(openapi.VectorDiff{
		Incoming: openapi.VectorContent{
			Ref:        promotion.Spec.Vector,
			Components: toVectorComponents(incomingVectorContent.Components),
			Config:     toVectorConfig(incomingVectorContent.Config),
		},
		Current: openapi.VectorContent{
			Ref:        stage.Stage.Spec.Vector,
			Components: toVectorComponents(currentVectorContent.Components),
			Config:     toVectorConfig(currentVectorContent.Config),
		},
	}), nil
}

func toVectorComponents(cs []pkgcontent.VectorComponent) []openapi.VectorComponent {
	out := make([]openapi.VectorComponent, len(cs))
	for i, c := range cs {
		out[i] = openapi.VectorComponent{
			ComponentName: c.ComponentName,
			Version:       c.Version,
		}
	}
	return out
}

func toVectorConfig(cfg *pkgcontent.VectorConfig) *openapi.VectorConfig {
	if cfg == nil {
		return nil
	}
	result := &openapi.VectorConfig{}
	if len(cfg.Features) > 0 {
		f := cfg.Features
		result.Features = &f
	}
	if len(cfg.Authored) > 0 {
		a := cfg.Authored
		result.Authored = &a
	}
	return result
}
