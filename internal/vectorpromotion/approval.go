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
	// ErrAlreadyApproved is returned when the promotion is already approved.
	ErrAlreadyApproved = errors.New("promotion is already approved")
)

// Approve marks a VectorPromotion approved on behalf of approvedBy. It is the
// single entry point for granting approvals (used by the konfidence API): it
// keeps the condition vocabulary, the derived `status.state`, and the
// optimistic-locking discipline in one place. Callers can map
// `ErrPromotionFinished` and `ErrAlreadyApproved` to client errors.
func Approve(ctx context.Context, c client.Client, key types.NamespacedName, approvedBy string) error {
	vectorPromotion := &konfidence.VectorPromotion{}
	if err := c.Get(ctx, key, vectorPromotion); err != nil {
		return fmt.Errorf("failed to fetch VectorPromotion %q in namespace %q: %w", key.Name, key.Namespace, err)
	}

	if promotion.IsTerminal(vectorPromotion) {
		return fmt.Errorf("%w: state is %q", ErrPromotionFinished, vectorPromotion.Status.State)
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
		Message:            fmt.Sprintf("approved by %s", approvedBy),
	})
	vectorPromotion.Status.State = promotion.DeriveState(vectorPromotion)

	patch := client.MergeFromWithOptions(original, client.MergeFromWithOptimisticLock{})
	if err := c.Status().Patch(ctx, vectorPromotion, patch); err != nil {
		return fmt.Errorf("failed to patch approval onto VectorPromotion %q in namespace %q: %w",
			key.Name, key.Namespace, err)
	}
	return nil
}
