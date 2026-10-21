package handler

import (
	"context"
	"errors"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/api/apierror"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	landscapedomain "github.com/konfidence-project/konfidence/internal/landscape"
)

func (h *projectHandler) GetLandscapeV1(ctx context.Context, req openapi.GetLandscapeV1RequestObject) (openapi.GetLandscapeV1ResponseObject, error) {
	_, namespace, err := h.resolveProjectRequest(ctx, req.ProjectId)
	if err != nil {
		return nil, err
	}

	landscape, err := h.landscapeRepo.Get(ctx, namespace, req.LandscapeId)
	if err != nil {
		if errors.Is(err, landscapedomain.ErrLandscapeNotFound) {
			return nil, apierror.NewNotFound("landscape", req.LandscapeId)
		}
		return nil, apierror.NewInternal(err)
	}

	return openapi.GetLandscapeV1200JSONResponse(toLandscapeResponse(*landscape)), nil
}

func (h *projectHandler) ListLandscapesV1(ctx context.Context, req openapi.ListLandscapesV1RequestObject) (openapi.ListLandscapesV1ResponseObject, error) {
	_, namespace, err := h.resolveProjectRequest(ctx, req.ProjectId)
	if err != nil {
		return nil, err
	}

	landscapes, err := h.landscapeRepo.ListForProject(ctx, namespace)
	if err != nil {
		return nil, apierror.NewInternal(err)
	}

	data := make([]openapi.Landscape, len(landscapes))
	for i, l := range landscapes {
		data[i] = toLandscapeResponse(l)
	}

	return openapi.ListLandscapesV1200JSONResponse{Data: data}, nil
}

func toLandscapeResponse(l konfidence.Landscape) openapi.Landscape {
	return openapi.Landscape{
		Id:   l.Name,
		Name: l.Spec.DisplayName,
	}
}

// can be reused for artifact deployment
func (h *projectHandler) resolveLandscapeScope(ctx context.Context,
	projectNamespace string, landscapeId string, scopeOpts ...landscapedomain.ScopeOption) ([]landscapedomain.ScopedLandscape, error) {
	scope, err := h.landscapeRepo.ResolveScope(ctx, projectNamespace, scopeOpts...)
	if err != nil {
		if errors.Is(err, landscapedomain.ErrLandscapeNotFound) {
			return nil, apierror.NewNotFound("landscape", landscapeId)
		}
		return nil, apierror.NewInternal(err)
	}

	return scope, nil
}
