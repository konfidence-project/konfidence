package handler

import (
	"context"
	"errors"
	"fmt"
	"reflect"

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

	leftVC, err := h.ocmReader.GetVectorContent(ctx, request.Params.Left)
	if err != nil {
		if errors.Is(err, pkgocm.ErrNotFound) {
			return nil, apierror.NewNotFound("left vector", request.Params.Left)
		}
		if errors.Is(err, kompref.ErrInvalidComponentReference) {
			return nil, apierror.NewBadRequest(fmt.Sprintf("invalid left ref: %s", err))
		}
		return nil, apierror.NewInternal(err)
	}

	rightVC, err := h.ocmReader.GetVectorContent(ctx, request.Params.Right)
	if err != nil {
		if errors.Is(err, pkgocm.ErrNotFound) {
			return nil, apierror.NewNotFound("right vector", request.Params.Right)
		}
		if errors.Is(err, kompref.ErrInvalidComponentReference) {
			return nil, apierror.NewBadRequest(fmt.Sprintf("invalid right ref: %s", err))
		}
		return nil, apierror.NewInternal(err)
	}

	includeUnchanged := request.Params.IncludeUnchanged != nil && *request.Params.IncludeUnchanged == "true"

	return openapi.GetVectorDiffV1200JSONResponse(openapi.VectorDiff{
		Left:       request.Params.Left,
		Right:      request.Params.Right,
		Components: calculateDiffComponents(leftVC.Components, rightVC.Components, includeUnchanged),
		Config:     calculateDiffConfig(leftVC.Config, rightVC.Config, includeUnchanged),
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

func calculateDiffComponents(left, right []pkgcontent.VectorComponent, includeUnchanged bool) []openapi.VectorComponentDiff {
	rightMap := make(map[string]string, len(right))
	for _, c := range right {
		rightMap[c.ComponentName] = c.Version
	}

	result := make([]openapi.VectorComponentDiff, 0)

	for _, c := range left {
		rv, inRight := rightMap[c.ComponentName]
		switch {
		case !inRight:
			lv := c.Version
			result = append(result, openapi.VectorComponentDiff{
				ComponentName: c.ComponentName,
				LeftVersion:   &lv,
				ChangeType:    openapi.Removed,
			})
		case c.Version != rv:
			lv := c.Version
			result = append(result, openapi.VectorComponentDiff{
				ComponentName: c.ComponentName,
				LeftVersion:   &lv,
				RightVersion:  &rv,
				ChangeType:    openapi.Changed,
			})
		default:
			if !includeUnchanged {
				continue
			}
			lv := c.Version
			result = append(result, openapi.VectorComponentDiff{
				ComponentName: c.ComponentName,
				LeftVersion:   &lv,
				RightVersion:  &rv,
				ChangeType:    openapi.Unchanged,
			})
		}
	}

	leftMap := make(map[string]string, len(left))
	for _, c := range left {
		leftMap[c.ComponentName] = c.Version
	}

	for _, c := range right {
		if _, inLeft := leftMap[c.ComponentName]; !inLeft {
			rv := c.Version
			result = append(result, openapi.VectorComponentDiff{
				ComponentName: c.ComponentName,
				RightVersion:  &rv,
				ChangeType:    openapi.Added,
			})
		}
	}

	return result
}

func calculateDiffConfig(left, right *pkgcontent.VectorConfig, includeUnchanged bool) *openapi.VectorConfigDiff {
	var leftFeatures, leftAuthored map[string]any
	var rightFeatures, rightAuthored map[string]any

	if left != nil {
		leftFeatures = left.Features
		leftAuthored = left.Authored
	}
	if right != nil {
		rightFeatures = right.Features
		rightAuthored = right.Authored
	}

	features := createDiffConfigMap(leftFeatures, rightFeatures, includeUnchanged)
	authored := createDiffConfigMap(leftAuthored, rightAuthored, includeUnchanged)
	return &openapi.VectorConfigDiff{
		Features: &features,
		Authored: &authored,
	}
}

func createDiffConfigMap(left, right map[string]any, includeUnchanged bool) []openapi.ConfigEntryDiff {
	result := make([]openapi.ConfigEntryDiff, 0)

	for k, lv := range left {
		rv, inRight := right[k]
		switch {
		case !inRight:
			result = append(result, openapi.ConfigEntryDiff{
				Key:        k,
				LeftValue:  &lv,
				ChangeType: openapi.Removed,
			})
		case !reflect.DeepEqual(lv, rv):
			result = append(result, openapi.ConfigEntryDiff{
				Key:        k,
				LeftValue:  &lv,
				RightValue: &rv,
				ChangeType: openapi.Changed,
			})
		default:
			if !includeUnchanged {
				continue
			}
			result = append(result, openapi.ConfigEntryDiff{
				Key:        k,
				LeftValue:  &lv,
				RightValue: &rv,
				ChangeType: openapi.Unchanged,
			})
		}
	}

	for k, rv := range right {
		if _, exists := left[k]; !exists {
			result = append(result, openapi.ConfigEntryDiff{
				Key:        k,
				RightValue: &rv,
				ChangeType: openapi.Added,
			})
		}
	}

	return result
}
