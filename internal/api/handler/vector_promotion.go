package handler

import (
	"context"
	"errors"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/api/apierror"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	vectorpromotiondomain "github.com/konfidence-project/konfidence/internal/vectorpromotion"
	"github.com/konfidence-project/konfidence/pkg/promotiongate"
	"k8s.io/apimachinery/pkg/api/meta"
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
	case errors.Is(err, vectorpromotiondomain.ErrPromotionFinished),
		errors.Is(err, vectorpromotiondomain.ErrPromotionExecuting),
		errors.Is(err, vectorpromotiondomain.ErrConflict),
		errors.Is(err, vectorpromotiondomain.ErrApprovalNotRequired):
		return nil, apierror.NewConflict(err.Error())
	default:
		return nil, apierror.NewInternal(err)
	}
}

func toVectorPromotionResponse(p konfidence.VectorPromotion) openapi.VectorPromotion {
	sequence := p.Spec.Sequence

	promotion := openapi.VectorPromotion{
		Id:       p.Name,
		Sequence: &sequence,
		Source:   toPromotionSourceReferenceResponse(p.Spec.Source),
		Target:   toPromotionTargetReferenceResponse(p.Spec.Target),
		Vector:   p.Spec.Vector,
	}

	status := displayState(&p)
	promotion.Status = &status

	if p.Spec.TTLAfterFinished != nil {
		ttl := p.Spec.TTLAfterFinished.Duration.String()
		promotion.TtlAfterFinished = &ttl
	}

	if len(p.Spec.Gates) > 0 {
		gates := toPromotionGateStatusesResponse(&p)
		promotion.Gates = &gates
	}

	return promotion
}

// displayState summarizes the promotion's observed facts into the single
// status shown by the API. It exists only for display; the controllers decide
// on the facts themselves. Condition reasons are not consulted.
func displayState(p *konfidence.VectorPromotion) openapi.VectorPromotionStatus {
	conditions := p.Status.Conditions
	switch {
	case meta.IsStatusConditionTrue(conditions, konfidence.ConditionTypeSucceeded):
		return openapi.VectorPromotionStatusSucceeded
	case meta.IsStatusConditionFalse(conditions, konfidence.ConditionTypeSucceeded):
		return openapi.VectorPromotionStatusSuperseded
	case p.Status.StartTime != nil:
		return openapi.VectorPromotionStatusInProgress
	case meta.IsStatusConditionFalse(conditions, konfidence.ConditionTypeTargetResolved):
		return openapi.VectorPromotionStatusBlocked
	case meta.IsStatusConditionTrue(conditions, konfidence.ConditionTypeGatesSatisfied):
		return openapi.VectorPromotionStatusReady
	default:
		return openapi.VectorPromotionStatusWaiting
	}
}

// toPromotionGateStatusesResponse reports every declared gate in declaration
// order; a declared gate without a status entry is reported as Pending.
func toPromotionGateStatusesResponse(p *konfidence.VectorPromotion) []openapi.PromotionGateStatus {
	gates := make([]openapi.PromotionGateStatus, 0, len(p.Spec.Gates))
	for _, declared := range p.Spec.Gates {
		gate := openapi.PromotionGateStatus{
			Type:  declared.Type,
			State: openapi.PromotionGateStatusState(konfidence.PromotionGateStatePending),
		}
		if entry := promotiongate.Find(p, declared.Type); entry != nil {
			gate.State = openapi.PromotionGateStatusState(entry.State)
			gate.Reason = optionalString(entry.Reason)
			gate.Message = optionalString(entry.Message)
			gate.SetBy = optionalString(entry.SetBy)
			if entry.SetAt != nil {
				setAt := entry.SetAt.Time
				gate.SetAt = &setAt
			}
		}
		gates = append(gates, gate)
	}
	return gates
}

func toPromotionGatesResponse(gates []konfidence.PromotionGate) []openapi.PromotionGate {
	result := make([]openapi.PromotionGate, 0, len(gates))
	for _, gate := range gates {
		result = append(result, openapi.PromotionGate{Type: gate.Type})
	}
	return result
}

func optionalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
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
