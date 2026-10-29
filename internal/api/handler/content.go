package handler

import (
	"context"
	"errors"
	"fmt"

	"github.com/konfidence-project/konfidence/internal/api/apierror"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	"github.com/konfidence-project/konfidence/internal/api/session"
	kompref "github.com/konfidence-project/konfidence/pkg/ocm/compref"
	pkgcontent "github.com/konfidence-project/konfidence/pkg/ocm/content"
	pkgocm "github.com/konfidence-project/konfidence/pkg/ocm/repository"
)

func (h *projectHandler) GetVectorContentV1(ctx context.Context,
	request openapi.GetVectorContentV1RequestObject) (openapi.GetVectorContentV1ResponseObject, error) {
	identity, err := session.FromContext(ctx)
	if err != nil {
		return nil, apierror.NewUnauthorized()
	}

	if _, err := h.resolveProjectNamespace(ctx, identity, request.ProjectId); err != nil {
		return nil, err
	}

	results := make([]openapi.VectorContent, 0, len(request.Params.Refs))
	for _, rawOcmRef := range request.Params.Refs {
		vectorContent, err := h.ocmReader.GetVectorContent(ctx, rawOcmRef)
		if err != nil {
			if errors.Is(err, kompref.ErrInvalidComponentReference) {
				return nil, apierror.NewBadRequest(fmt.Sprintf("invalid ref %q: %s", rawOcmRef, err))
			}
			if errors.Is(err, pkgocm.ErrNotFound) {
				return nil, apierror.NewNotFound("vector", rawOcmRef)
			}
			return nil, apierror.NewInternal(err)
		}

		results = append(results, openapi.VectorContent{
			Ref:        rawOcmRef,
			Components: toVectorComponents(vectorContent.Components),
			Config:     toVectorConfig(vectorContent.Config),
		})
	}

	return openapi.GetVectorContentV1200JSONResponse(openapi.VectorContentList{
		Data: results,
	}), nil
}
func (h *projectHandler) GetArtifactContentV1(ctx context.Context,
	request openapi.GetArtifactContentV1RequestObject) (openapi.GetArtifactContentV1ResponseObject, error) {
	identity, err := session.FromContext(ctx)
	if err != nil {
		return nil, apierror.NewUnauthorized()
	}

	if _, err := h.resolveProjectNamespace(ctx, identity, request.ProjectId); err != nil {
		return nil, err
	}

	artifactContent, err := h.ocmReader.GetArtifactContent(ctx, request.Params.Ref)
	if err != nil {
		if errors.Is(err, kompref.ErrInvalidComponentReference) {
			return nil, apierror.NewBadRequest(fmt.Sprintf("invalid ref %q: %s", request.Params.Ref, err))
		}
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

func (h *projectHandler) GetVectorDiffV1(ctx context.Context,
	request openapi.GetVectorDiffV1RequestObject) (openapi.GetVectorDiffV1ResponseObject, error) {
	identity, err := session.FromContext(ctx)
	if err != nil {
		return nil, apierror.NewUnauthorized()
	}

	if _, err := h.resolveProjectNamespace(ctx, identity, request.ProjectId); err != nil {
		return nil, err
	}

	incomingVC, err := h.ocmReader.GetVectorContent(ctx, request.Params.Incoming)
	if err != nil {
		if errors.Is(err, pkgocm.ErrNotFound) {
			return nil, apierror.NewNotFound("incoming vector", request.Params.Incoming)
		}
		if errors.Is(err, kompref.ErrInvalidComponentReference) {
			return nil, apierror.NewBadRequest(fmt.Sprintf("invalid incoming ref: %s", err))
		}
		return nil, apierror.NewInternal(err)
	}

	currentVC, err := h.ocmReader.GetVectorContent(ctx, request.Params.Current)
	if err != nil {
		if errors.Is(err, pkgocm.ErrNotFound) {
			return nil, apierror.NewNotFound("current vector", request.Params.Current)
		}
		if errors.Is(err, kompref.ErrInvalidComponentReference) {
			return nil, apierror.NewBadRequest(fmt.Sprintf("invalid current ref: %s", err))
		}
		return nil, apierror.NewInternal(err)
	}

	return openapi.GetVectorDiffV1200JSONResponse(openapi.VectorDiff{
		Incoming: openapi.VectorContent{
			Ref:        request.Params.Incoming,
			Components: toVectorComponents(incomingVC.Components),
			Config:     toVectorConfig(incomingVC.Config),
		},
		Current: openapi.VectorContent{
			Ref:        request.Params.Current,
			Components: toVectorComponents(currentVC.Components),
			Config:     toVectorConfig(currentVC.Config),
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
