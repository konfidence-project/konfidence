package vectorpromotion

import (
	"context"
	"errors"
	"fmt"
	"sort"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/konfidence-project/konfidence/internal/vectorpromotion/internal/promotion"
)

// PromotionConfigNameField is the field-index key that maps a VectorPromotion to its
// VectorPromotionConfig. Register this index on any informer cache or fake client that
// backs ListForConfig queries
const PromotionConfigNameField = "spec.vectorPromotionConfigName"

// PromotionConfigNameIndexFunc extracts the VectorPromotionConfig name from a
// VectorPromotion for field-index registration.
func PromotionConfigNameIndexFunc(obj client.Object) []string {
	vp, ok := obj.(*konfidence.VectorPromotion)
	if !ok {
		return nil
	}
	return []string{vp.Spec.VectorPromotionConfigName}
}

var (
	// ErrVectorPromotionNotFound is returned when the VectorPromotion does not exist.
	ErrVectorPromotionNotFound = errors.New("vectorPromotion not found")
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

type Repository interface {
	Get(ctx context.Context, namespace string, vectorPromotionId string) (*konfidence.VectorPromotion, error)
	ListForConfig(ctx context.Context, namespace string, vectorPromotionConfigName string) ([]konfidence.VectorPromotion, error)
	Approve(ctx context.Context, namespace string, vectorPromotionId string, approvedBy string) error
}

type k8sRepository struct{ k8sClient client.Client }

func NewRepository(k8sClient client.Client) Repository {
	return &k8sRepository{k8sClient: k8sClient}
}

func (r *k8sRepository) Get(ctx context.Context, namespace string, vectorPromotionId string) (*konfidence.VectorPromotion, error) {
	var vectorPromotion konfidence.VectorPromotion
	if err := r.k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: vectorPromotionId}, &vectorPromotion); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, ErrVectorPromotionNotFound
		}
		return nil, fmt.Errorf("getting VectorPromotion %q failed: %w", vectorPromotionId, err)
	}

	return &vectorPromotion, nil
}

// ListForConfig returns every VectorPromotion in the namespace that belongs to the given
// config, ordered by their monotonic sequence. The lookup is backed by the
// PromotionConfigNameField index; register it on any cache or fake client that backs this
// repository (see PromotionConfigNameIndexFunc).
func (r *k8sRepository) ListForConfig(ctx context.Context, namespace string,
	vectorPromotionConfigName string) ([]konfidence.VectorPromotion, error) {
	var list konfidence.VectorPromotionList
	if err := r.k8sClient.List(ctx, &list,
		client.InNamespace(namespace),
		client.MatchingFields{PromotionConfigNameField: vectorPromotionConfigName}); err != nil {
		return nil, fmt.Errorf("listing VectorPromotions in namespace %q failed: %w", namespace, err)
	}

	sort.Slice(list.Items, func(i, j int) bool {
		return list.Items[i].Spec.Sequence < list.Items[j].Spec.Sequence
	})

	return list.Items, nil
}

// Approve marks a VectorPromotion approved on behalf of approvedBy. It is the
// single entry point for granting approvals (used by the konfidence API): it
// keeps the condition vocabulary, the derived `status.state`, and the
// optimistic-locking discipline in one place. A missing resource is reported as
// ErrVectorPromotionNotFound so callers can map it the same way they map a
// failed Get; callers can map ErrPromotionSuperseded, ErrPromotionFinished,
// ErrAlreadyApproved, ErrApprovalNotRequired and ErrApproverMissing to client
// errors.
func (r *k8sRepository) Approve(ctx context.Context, namespace, vectorPromotionId, approvedBy string) error {
	if approvedBy == "" {
		return ErrApproverMissing
	}

	key := types.NamespacedName{Namespace: namespace, Name: vectorPromotionId}
	vectorPromotion := &konfidence.VectorPromotion{}
	if err := r.k8sClient.Get(ctx, key, vectorPromotion); err != nil {
		if apierrors.IsNotFound(err) {
			return ErrVectorPromotionNotFound
		}
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
	if err := r.k8sClient.Status().Patch(ctx, vectorPromotion, patch); err != nil {
		return fmt.Errorf("failed to patch approval onto VectorPromotion %q in namespace %q: %w",
			key.Name, key.Namespace, err)
	}

	return nil
}
