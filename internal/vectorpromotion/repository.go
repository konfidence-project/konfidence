package vectorpromotion

import (
	"context"
	"errors"
	"fmt"
	"sort"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/konfidence-project/konfidence/internal/vectorpromotion/internal/promotion"
	"github.com/konfidence-project/konfidence/pkg/promotiongate"
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
	// ErrPromotionExecuting is returned when the promotion already started
	// executing, so its gates can no longer change.
	ErrPromotionExecuting = errors.New("promotion is already executing; its gates can no longer change")
	// ErrAlreadyApproved is returned when the explicit-approval gate is
	// already satisfied.
	ErrAlreadyApproved = errors.New("promotion is already approved")
	// ErrApprovalNotRequired is returned when the promotion does not declare
	// the explicit-approval gate: there is nothing to approve.
	ErrApprovalNotRequired = errors.New("promotion has no explicit-approval gate; nothing to approve")
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
// config, ordered by their monotonic sequence descending (newest first). The lookup is
// backed by the PromotionConfigNameField index; register it on any cache or fake client
// that backs this repository (see PromotionConfigNameIndexFunc).
func (r *k8sRepository) ListForConfig(ctx context.Context, namespace string,
	vectorPromotionConfigName string) ([]konfidence.VectorPromotion, error) {
	var list konfidence.VectorPromotionList
	if err := r.k8sClient.List(ctx, &list,
		client.InNamespace(namespace),
		client.MatchingFields{PromotionConfigNameField: vectorPromotionConfigName}); err != nil {
		return nil, fmt.Errorf("listing VectorPromotions in namespace %q failed: %w", namespace, err)
	}

	sort.Slice(list.Items, func(i, j int) bool {
		return list.Items[i].Spec.Sequence > list.Items[j].Spec.Sequence
	})

	return list.Items, nil
}

// Approve satisfies the promotion's `explicit-approval.konfidence.cloud` gate
// on behalf of approvedBy. It is the single entry point for human approvals
// (used by the konfidence API) and writes through pkg/promotiongate, like any
// other gate writer, so state and provenance are set together under optimistic
// locking. The derived `status.state` follows once the promotion controller
// observes the gate change.
//
// A missing resource is reported as ErrVectorPromotionNotFound; callers can map
// ErrPromotionSuperseded, ErrPromotionFinished, ErrPromotionExecuting,
// ErrAlreadyApproved, ErrApprovalNotRequired and ErrApproverMissing to client
// errors.
func (r *k8sRepository) Approve(ctx context.Context, namespace, vectorPromotionId, approvedBy string) error {
	if approvedBy == "" {
		return ErrApproverMissing
	}

	vectorPromotion, err := r.Get(ctx, namespace, vectorPromotionId)
	if err != nil {
		return err
	}

	if promotion.IsSuperseded(vectorPromotion) {
		return ErrPromotionSuperseded
	}
	if promotion.IsTerminal(vectorPromotion) {
		return fmt.Errorf("%w: state is %q", ErrPromotionFinished, vectorPromotion.Status.State)
	}
	if !promotiongate.Declared(vectorPromotion, konfidence.GateTypeExplicitApproval) {
		return ErrApprovalNotRequired
	}
	if promotiongate.StateOf(vectorPromotion, konfidence.GateTypeExplicitApproval) == konfidence.PromotionGateStateSatisfied {
		return ErrAlreadyApproved
	}

	key := types.NamespacedName{Namespace: namespace, Name: vectorPromotionId}
	err = promotiongate.Set(ctx, r.k8sClient, key, promotiongate.Update{
		Type:    konfidence.GateTypeExplicitApproval,
		State:   konfidence.PromotionGateStateSatisfied,
		Reason:  konfidence.ReasonGateApprovedByUser,
		Message: fmt.Sprintf("approved by %s", approvedBy),
		SetBy:   approvedBy,
	})
	switch {
	case err == nil:
		return nil
	case errors.Is(err, promotiongate.ErrNotFound):
		return ErrVectorPromotionNotFound
	case errors.Is(err, promotiongate.ErrGateNotDeclared):
		return ErrApprovalNotRequired
	case errors.Is(err, promotiongate.ErrPromotionNotMutable):
		return ErrPromotionExecuting
	default:
		return fmt.Errorf("failed to approve VectorPromotion %q in namespace %q: %w", key.Name, key.Namespace, err)
	}
}
