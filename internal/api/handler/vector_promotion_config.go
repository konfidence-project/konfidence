package handler

import (
	"context"
	"errors"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/api/apierror"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	vectorpromotiondomain "github.com/konfidence-project/konfidence/internal/vectorpromotion"
)

func (h *projectHandler) GetVectorPromotionConfigV1(ctx context.Context,
	request openapi.GetVectorPromotionConfigV1RequestObject) (openapi.GetVectorPromotionConfigV1ResponseObject, error) {
	_, namespace, err := h.resolveProjectRequest(ctx, request.ProjectId)
	if err != nil {
		return nil, err
	}

	config, err := h.vectorPromotionConfigRepo.Get(ctx, namespace, request.VectorPromotionConfigId)
	if err != nil {
		if errors.Is(err, vectorpromotiondomain.ErrVectorPromotionConfigNotFound) {
			return nil, apierror.NewNotFound("vectorPromotionConfig", request.VectorPromotionConfigId)
		}
		return nil, apierror.NewInternal(err)
	}

	promotions, err := h.vectorPromotionRepo.ListForConfig(ctx, namespace, config.Name)
	if err != nil {
		return nil, apierror.NewInternal(err)
	}

	response := toVectorPromotionConfigResponse(*config, promotions)
	return openapi.GetVectorPromotionConfigV1200JSONResponse(response), nil
}

func (h *projectHandler) ListVectorPromotionConfigsV1(ctx context.Context,
	request openapi.ListVectorPromotionConfigsV1RequestObject) (openapi.ListVectorPromotionConfigsV1ResponseObject, error) {
	_, namespace, err := h.resolveProjectRequest(ctx, request.ProjectId)
	if err != nil {
		return nil, err
	}

	configs, err := h.vectorPromotionConfigRepo.List(ctx, namespace)
	if err != nil {
		return nil, apierror.NewInternal(err)
	}

	data := make([]openapi.VectorPromotionConfig, 0, len(configs))
	for _, c := range configs {
		promotions, err := h.vectorPromotionRepo.ListForConfig(ctx, namespace, c.Name)
		if err != nil {
			return nil, apierror.NewInternal(err)
		}
		data = append(data, toVectorPromotionConfigResponse(c, promotions))
	}

	return openapi.ListVectorPromotionConfigsV1200JSONResponse{Data: data}, nil
}

func toVectorPromotionConfigResponse(c konfidence.VectorPromotionConfig,
	promotions []konfidence.VectorPromotion) openapi.VectorPromotionConfig {
	config := openapi.VectorPromotionConfig{
		Id:     c.Name,
		Source: toPromotionSourceReferenceResponse(c.Spec.Source),
		Target: toPromotionTargetReferenceResponse(c.Spec.Target),
	}

	if len(c.Spec.Gates) > 0 {
		gates := toPromotionGatesResponse(c.Spec.Gates)
		config.Gates = &gates
	}

	if c.Spec.TTLAfterFinished != nil {
		ttl := c.Spec.TTLAfterFinished.Duration.String()
		config.TtlAfterFinished = &ttl
	}

	if c.Spec.KeepLastPromotions != nil {
		keepLast := int(*c.Spec.KeepLastPromotions)
		config.KeepLastPromotions = &keepLast
	}

	config.Promotions = make([]openapi.VectorPromotion, len(promotions))
	for i, promotion := range promotions {
		config.Promotions[i] = toVectorPromotionResponse(promotion)
	}

	return config
}
