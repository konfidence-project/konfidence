package vectorpromotion

import (
	"context"
	"errors"
	"fmt"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/konfidence-project/konfidence/internal/vectorpromotion/internal/promotion"
)

var (
	// ErrPromotionFinished is returned when the promotion already reached a
	// terminal state and can no longer be approved.
	ErrPromotionFinished = errors.New("promotion already reached a terminal state")
	// ErrPromotionSuperseded is returned when the promotion was superseded by
	// a newer promotion. Superseded promotions are locked for good: the newer
	// promotion is the one to approve.
	ErrPromotionSuperseded = errors.New("promotion was superseded and is locked")
	// ErrAlreadyApproved is returned when the promotion is already approved.
	ErrAlreadyApproved = errors.New("promotion is already approved")
	// ErrApprovalNotRequired is returned when the promotion has no approval
	// gate: there is nothing to approve.
	ErrApprovalNotRequired = errors.New("promotion does not require approval; nothing to approve")
	// ErrApproverMissing is returned when no approver identity was supplied:
	// an approval without provenance is not an audit record.
	ErrApproverMissing = errors.New("approvedBy must not be empty")
)

// Approve marks a VectorPromotion approved on behalf of approvedBy. It is the
// single entry point for granting approvals (used by the konfidence API): it
// keeps the condition vocabulary, the derived `status.state`, and the
// optimistic-locking discipline in one place. Callers can map
// `ErrPromotionSuperseded`, `ErrPromotionFinished`, `ErrAlreadyApproved`,
// `ErrApprovalNotRequired` and `ErrApproverMissing` to client errors.
func Approve(ctx context.Context, c client.Client, key types.NamespacedName, approvedBy string) error {
	if approvedBy == "" {
		return ErrApproverMissing
	}

	vectorPromotion := &konfidence.VectorPromotion{}
	if err := c.Get(ctx, key, vectorPromotion); err != nil {
		return fmt.Errorf("failed to fetch VectorPromotion %q in namespace %q: %w", key.Name, key.Namespace, err)
	}

	if promotion.IsSuperseded(vectorPromotion) {
		return ErrPromotionSuperseded
	}
	if promotion.IsTerminal(vectorPromotion) {
		return fmt.Errorf("%w: state is %q", ErrPromotionFinished, vectorPromotion.Status.State)
	}
	if !vectorPromotion.Spec.RequireApproval {
		return ErrApprovalNotRequired
	}
	if promotion.IsApproved(vectorPromotion) {
		return ErrAlreadyApproved
	}

	original := vectorPromotion.DeepCopy()
	meta.SetStatusCondition(&vectorPromotion.Status.Conditions, metav1.Condition{
		Type:               konfidence.ConditionTypeApproved,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: vectorPromotion.Generation,
		Reason:             konfidence.ReasonPromotionManuallyApproved,
		Message:            fmt.Sprintf("promotion of vector %q approved by %s", vectorPromotion.Spec.Vector, approvedBy),
	})
	vectorPromotion.Status.Approval = &konfidence.PromotionApproval{
		ApprovedBy: approvedBy,
		ApprovedAt: metav1.Now(),
	}
	vectorPromotion.Status.State = promotion.DeriveState(vectorPromotion)

	patch := client.MergeFromWithOptions(original, client.MergeFromWithOptimisticLock{})
	if err := c.Status().Patch(ctx, vectorPromotion, patch); err != nil {
		return fmt.Errorf("failed to patch approval onto VectorPromotion %q in namespace %q: %w",
			key.Name, key.Namespace, err)
	}
	return nil
}
