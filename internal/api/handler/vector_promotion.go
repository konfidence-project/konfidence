package handler

import (
	"context"
	"errors"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/api/apierror"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	vectorpromotiondomain "github.com/konfidence-project/konfidence/internal/vectorpromotion"
)

func (h *projectHandler) GetVectorPromotionV1(ctx context.Context,
	request openapi.GetVectorPromotionV1RequestObject) (openapi.GetVectorPromotionV1ResponseObject, error) {
	_, namespace, err := h.resolveProjectRequest(ctx, request.ProjectId)
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

	response := toVectorPromotionResponse(*promotion)
	return openapi.GetVectorPromotionV1200JSONResponse(response), nil
}

func (h *projectHandler) ApproveVectorPromotionV1(ctx context.Context,
	request openapi.ApproveVectorPromotionV1RequestObject) (openapi.ApproveVectorPromotionV1ResponseObject, error) {
	identity, namespace, err := h.resolveProjectRequest(ctx, request.ProjectId)
	if err != nil {
		return nil, err
	}

	err = h.vectorPromotionRepo.Approve(ctx, namespace, request.VectorPromotionId, identity.Subject)
	switch {
	case err == nil, errors.Is(err, vectorpromotiondomain.ErrAlreadyApproved):
		// Approving twice is idempotent: the promotion is already approved,
		// so report success rather than a conflict.
		return openapi.ApproveVectorPromotionV1204Response{}, nil
	case errors.Is(err, vectorpromotiondomain.ErrVectorPromotionNotFound):
		return nil, apierror.NewNotFound("vectorPromotion", request.VectorPromotionId)
	case errors.Is(err, vectorpromotiondomain.ErrPromotionSuperseded),
		errors.Is(err, vectorpromotiondomain.ErrPromotionFinished),
		errors.Is(err, vectorpromotiondomain.ErrApprovalNotRequired):
		return nil, apierror.NewConflict(err.Error())
	default:
		return nil, apierror.NewInternal(err)
	}
}

func toVectorPromotionResponse(p konfidence.VectorPromotion) openapi.VectorPromotion {
	requireApproval := p.Spec.RequireApproval
	sequence := p.Spec.Sequence

	promotion := openapi.VectorPromotion{
		Id:              p.Name,
		RequireApproval: &requireApproval,
		Sequence:        &sequence,
		Source:          toPromotionSourceReferenceResponse(p.Spec.Source),
		Target:          toPromotionTargetReferenceResponse(p.Spec.Target),
		Vector:          p.Spec.Vector,
	}

	if p.Status.State != "" {
		status := openapi.VectorPromotionStatus(p.Status.State)
		promotion.Status = &status
	}

	if p.Spec.TTLAfterFinished != nil {
		ttl := p.Spec.TTLAfterFinished.Duration.String()
		promotion.TtlAfterFinished = &ttl
	}

	if p.Status.Approval != nil {
		promotion.Approval = &openapi.PromotionApproval{
			ApprovedBy: p.Status.Approval.ApprovedBy,
			ApprovedAt: p.Status.Approval.ApprovedAt.Time,
		}
	}

	return promotion
}

func toPromotionSourceReferenceResponse(s konfidence.PromotionSourceReference) openapi.PromotionSourceReference {
	source := openapi.PromotionSourceReference{
		Kind: openapi.PromotionSourceReferenceKind(s.Kind),
		Name: s.Name,
	}
	if s.Landscape != "" {
		landscape := s.Landscape
		source.Landscape = &landscape
	}
	return source
}

func toPromotionTargetReferenceResponse(t konfidence.PromotionTargetReference) openapi.PromotionTargetReference {
	return openapi.PromotionTargetReference{
		Kind:      openapi.PromotionTargetReferenceKind(t.Kind),
		Name:      t.Name,
		Landscape: t.Landscape,
	}
}
