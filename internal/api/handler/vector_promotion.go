package handler

import (
	"context"
	"errors"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/api/apierror"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	vectorpromotiondomain "github.com/konfidence-project/konfidence/internal/vectorpromotion"
	"github.com/konfidence-project/konfidence/pkg/promotiongate"
	"github.com/konfidence-project/konfidence/pkg/util"
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

	if err := h.vectorPromotionRepo.Approve(ctx, namespace, request.VectorPromotionId, identity.Subject); err != nil {
		return nil, promotionActionError(err, request.VectorPromotionId)
	}
	return openapi.ApproveVectorPromotionV1204Response{}, nil
}

func (h *projectHandler) RejectVectorPromotionV1(ctx context.Context,
	request openapi.RejectVectorPromotionV1RequestObject) (openapi.RejectVectorPromotionV1ResponseObject, error) {
	identity, namespace, err := h.resolveProjectRequest(ctx, request.ProjectId)
	if err != nil {
		return nil, err
	}
	if request.Body == nil {
		return nil, apierror.NewBadRequest("request body with a reason is required")
	}

	err = h.vectorPromotionRepo.Reject(ctx, namespace, request.VectorPromotionId, identity.Subject, request.Body.Reason)
	if err != nil {
		return nil, promotionActionError(err, request.VectorPromotionId)
	}
	return openapi.RejectVectorPromotionV1204Response{}, nil
}

func (h *projectHandler) BypassVectorPromotionGatesV1(ctx context.Context,
	request openapi.BypassVectorPromotionGatesV1RequestObject) (openapi.BypassVectorPromotionGatesV1ResponseObject, error) {
	identity, namespace, err := h.resolveProjectRequest(ctx, request.ProjectId)
	if err != nil {
		return nil, err
	}
	if request.Body == nil {
		return nil, apierror.NewBadRequest("request body with a reason is required")
	}

	err = h.vectorPromotionRepo.BypassGates(ctx, namespace, request.VectorPromotionId, identity.Subject, request.Body.Reason)
	if err != nil {
		return nil, promotionActionError(err, request.VectorPromotionId)
	}
	return openapi.BypassVectorPromotionGatesV1204Response{}, nil
}

// promotionActionError maps a failed approve, reject or gate bypass to its API error.
func promotionActionError(err error, vectorPromotionId string) error {
	switch {
	case errors.Is(err, vectorpromotiondomain.ErrVectorPromotionNotFound):
		return apierror.NewNotFound("vectorPromotion", vectorPromotionId)
	case errors.Is(err, vectorpromotiondomain.ErrReasonInvalid):
		return apierror.NewBadRequest(err.Error())
	case errors.Is(err, vectorpromotiondomain.ErrPromotionFinished),
		errors.Is(err, vectorpromotiondomain.ErrPromotionExecuting),
		errors.Is(err, vectorpromotiondomain.ErrPromotionOverridden),
		errors.Is(err, vectorpromotiondomain.ErrApprovalNotRequired),
		errors.Is(err, vectorpromotiondomain.ErrConflict):
		return apierror.NewConflict(err.Error())
	default:
		return apierror.NewInternal(err)
	}
}

func (h *projectHandler) RevokeApprovalVectorPromotionV1(ctx context.Context,
	request openapi.RevokeApprovalVectorPromotionV1RequestObject,
) (openapi.RevokeApprovalVectorPromotionV1ResponseObject, error) {
	identity, namespace, err := h.resolveProjectRequest(ctx, request.ProjectId)
	if err != nil {
		return nil, err
	}

	err = h.vectorPromotionRepo.RevokeApproval(ctx, namespace, request.VectorPromotionId, identity.Subject)
	switch {
	case err == nil:
		return openapi.RevokeApprovalVectorPromotionV1204Response{}, nil
	case errors.Is(err, vectorpromotiondomain.ErrVectorPromotionNotFound):
		return nil, apierror.NewNotFound("vectorPromotion", request.VectorPromotionId)
	case errors.Is(err, vectorpromotiondomain.ErrPromotionFinished),
		errors.Is(err, vectorpromotiondomain.ErrPromotionExecuting),
		errors.Is(err, vectorpromotiondomain.ErrNotApproved),
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

	if o := p.Status.Override; o != nil {
		promotion.Override = &openapi.PromotionOverride{
			Type:   openapi.PromotionOverrideType(o.Type),
			SetBy:  o.SetBy,
			SetAt:  o.SetAt.Time,
			Reason: o.Reason,
		}
	}

	return promotion
}

// displayState summarizes the promotion's observed facts into the single
// status shown by the API. An unresolved target takes precedence over
// InProgress: once execution can no longer resolve its target the promotion is
// stuck, so it reads as Blocked even if it had already started. A rejection
// wins over a concurrent supersede: either way the promotion never ran.
func displayState(p *konfidence.VectorPromotion) openapi.VectorPromotionStatus {
	conditions := p.Status.Conditions

	succeeded := meta.IsStatusConditionTrue(conditions, konfidence.ConditionTypeSucceeded)
	failed := meta.IsStatusConditionFalse(conditions, konfidence.ConditionTypeSucceeded)
	finished := succeeded || failed
	rejected := p.Status.Override != nil && p.Status.Override.Type == konfidence.PromotionOverrideTypeReject
	bypassed := p.Status.Override != nil && p.Status.Override.Type == konfidence.PromotionOverrideTypeBypassGates
	started := p.Status.StartTime != nil
	targetUnresolved := meta.IsStatusConditionFalse(conditions, konfidence.ConditionTypeTargetResolved)
	gatesSatisfied := meta.IsStatusConditionTrue(conditions, konfidence.ConditionTypeGatesSatisfied)

	switch {
	case succeeded:
		return openapi.VectorPromotionStatusSucceeded
	case failed && rejected:
		return openapi.VectorPromotionStatusRejected
	case failed:
		return openapi.VectorPromotionStatusSuperseded
	case !finished && targetUnresolved:
		return openapi.VectorPromotionStatusBlocked
	case !finished && started:
		return openapi.VectorPromotionStatusInProgress
	case !finished && (gatesSatisfied || bypassed):
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
			gate.Reason = util.OptionalString(entry.Reason)
			gate.Message = util.OptionalString(entry.Message)
			gate.SetBy = util.OptionalString(entry.SetBy)
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

func toPromotionSourceReferenceResponse(s konfidence.PromotionSourceReference) openapi.PromotionSourceReference {
	source := openapi.PromotionSourceReference{
		Kind: openapi.PromotionSourceReferenceKind(s.Kind),
		Name: s.Name,
	}
	source.Landscape = util.OptionalString(s.Landscape)
	return source
}

func toPromotionTargetReferenceResponse(t konfidence.PromotionTargetReference) openapi.PromotionTargetReference {
	return openapi.PromotionTargetReference{
		Kind:      openapi.PromotionTargetReferenceKind(t.Kind),
		Name:      t.Name,
		Landscape: t.Landscape,
	}
}
