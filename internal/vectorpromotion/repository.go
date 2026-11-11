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
	// ErrPromotionFinished is returned when the promotion already finished
	// (succeeded or superseded) and can no longer be approved. The error
	// carries the reason and message of the Succeeded condition.
	ErrPromotionFinished = errors.New("promotion already finished")
	// ErrPromotionExecuting is returned when the promotion's gates are
	// already satisfied, so they can no longer change.
	ErrPromotionExecuting = errors.New("promotion gates are already satisfied; they can no longer change")
	// ErrNotApproved is returned when there is no satisfied approval to revoke.
	ErrNotApproved = errors.New("promotion is not approved; nothing to revoke")
	// ErrApprovalNotRequired is returned when the promotion does not declare
	// the explicit-approval gate: there is nothing to approve.
	ErrApprovalNotRequired = errors.New("promotion has no explicit-approval gate; nothing to approve")
	// ErrConflict is returned when the write raced a concurrent write to the
	// promotion. The write did not happen; the caller should retry.
	ErrConflict = errors.New("promotion was modified concurrently; retry the request")
	// ErrPromotionOverridden is returned when a different override is already recorded.
	ErrPromotionOverridden = errors.New("promotion was already overridden by an operator")
	// ErrActorMissing is returned when no acting identity was supplied: a
	// decision without provenance is not an audit record.
	ErrActorMissing = errors.New("the acting identity must not be empty")
	// ErrReasonInvalid is returned for an empty or too long reason.
	ErrReasonInvalid = fmt.Errorf("reason must be between 1 and %d characters", MaxOverrideReasonLength)
)

// MaxOverrideReasonLength matches the CRD validation of `status.override.reason`.
const MaxOverrideReasonLength = 1024

type Repository interface {
	Get(ctx context.Context, namespace string, vectorPromotionId string) (*konfidence.VectorPromotion, error)
	ListForConfig(ctx context.Context, namespace string, vectorPromotionConfigName string) ([]konfidence.VectorPromotion, error)
	Approve(ctx context.Context, namespace string, vectorPromotionId string, approvedBy string) error
	RevokeApproval(ctx context.Context, namespace string, vectorPromotionId string, revokedBy string) error
	Reject(ctx context.Context, namespace string, vectorPromotionId string, rejectedBy string, reason string) error
	BypassGates(ctx context.Context, namespace string, vectorPromotionId string, bypassedBy string, reason string) error
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
// locking. The promotion controller sets GatesSatisfied True once it observes
// every gate satisfied, after which it never changes. Approving an approved
// promotion is a no-op.
//
// A missing resource is reported as ErrVectorPromotionNotFound; callers can map
// ErrPromotionFinished, ErrPromotionOverridden, ErrPromotionExecuting,
// ErrApprovalNotRequired, ErrActorMissing and ErrConflict to client errors.
func (r *k8sRepository) Approve(ctx context.Context, namespace, vectorPromotionId, approvedBy string) error {
	if approvedBy == "" {
		return ErrActorMissing
	}

	vectorPromotion, err := r.Get(ctx, namespace, vectorPromotionId)
	if err != nil {
		return err
	}

	if promotiongate.StateOf(vectorPromotion, konfidence.GateTypeExplicitApproval) == konfidence.PromotionGateStateSatisfied {
		return nil
	}
	if promotion.IsTerminal(vectorPromotion) {
		return finishedError(vectorPromotion)
	}
	if override := vectorPromotion.Status.Override; override != nil {
		return overriddenError(override)
	}

	key := types.NamespacedName{Namespace: namespace, Name: vectorPromotionId}
	err = promotiongate.Set(ctx, r.k8sClient, key, promotiongate.Update{
		Type:    konfidence.GateTypeExplicitApproval,
		State:   konfidence.PromotionGateStateSatisfied,
		Reason:  konfidence.ReasonGateApprovedByUser,
		Message: fmt.Sprintf("approved by %s", approvedBy),
		SetBy:   approvedBy,
	})
	return mapGateSetError(err, key, "approve")
}

// RevokeApproval sets the promotion's explicit-approval gate back to Pending
// on behalf of revokedBy. Repeating a recorded revocation is an idempotent
// success that preserves its provenance; other unsatisfied gates cannot be revoked.
// Revocation is rejected once the promotion has passed its gates, finished, or
// been overridden.
func (r *k8sRepository) RevokeApproval(ctx context.Context, namespace, vectorPromotionId, revokedBy string) error {
	if revokedBy == "" {
		return ErrActorMissing
	}

	vectorPromotion, err := r.Get(ctx, namespace, vectorPromotionId)
	if err != nil {
		return err
	}

	if promotion.IsTerminal(vectorPromotion) {
		return finishedError(vectorPromotion)
	}
	if override := vectorPromotion.Status.Override; override != nil {
		return overriddenError(override)
	}
	if !promotiongate.Declared(vectorPromotion, konfidence.GateTypeExplicitApproval) {
		return ErrApprovalNotRequired
	}
	if !promotiongate.GatesMutable(vectorPromotion) {
		return ErrPromotionExecuting
	}
	gate := promotiongate.Find(vectorPromotion, konfidence.GateTypeExplicitApproval)
	if gate != nil && gate.State == konfidence.PromotionGateStatePending &&
		gate.Reason == konfidence.ReasonGateApprovalRevokedByUser {
		return nil
	}
	if promotiongate.StateOf(vectorPromotion, konfidence.GateTypeExplicitApproval) != konfidence.PromotionGateStateSatisfied {
		return ErrNotApproved
	}

	key := types.NamespacedName{Namespace: namespace, Name: vectorPromotionId}
	err = promotiongate.Set(ctx, r.k8sClient, key, promotiongate.Update{
		Type:    konfidence.GateTypeExplicitApproval,
		State:   konfidence.PromotionGateStatePending,
		Reason:  konfidence.ReasonGateApprovalRevokedByUser,
		Message: fmt.Sprintf("approval revoked by %s", revokedBy),
		SetBy:   revokedBy,
	})
	return mapGateSetError(err, key, "revoke approval of")
}

// Reject records a `Reject` override on a promotion that still waits on its
// gates; the controller then finishes it as rejected.
func (r *k8sRepository) Reject(ctx context.Context, namespace, vectorPromotionId, rejectedBy, reason string) error {
	return r.setOverride(ctx, namespace, vectorPromotionId, konfidence.PromotionOverride{
		Type:   konfidence.PromotionOverrideTypeReject,
		SetBy:  rejectedBy,
		Reason: reason,
	})
}

// BypassGates records a `BypassGates` override on a promotion that still waits
// on its gates; the controller then proceeds as if all gates were satisfied.
func (r *k8sRepository) BypassGates(ctx context.Context, namespace, vectorPromotionId, bypassedBy, reason string) error {
	return r.setOverride(ctx, namespace, vectorPromotionId, konfidence.PromotionOverride{
		Type:   konfidence.PromotionOverrideTypeBypassGates,
		SetBy:  bypassedBy,
		Reason: reason,
	})
}

// setOverride patches with an optimistic lock so it cannot race the controller
// setting GatesSatisfied: whichever write lands second conflicts. Repeating the
// recorded decision is a no-op.
func (r *k8sRepository) setOverride(ctx context.Context, namespace, vectorPromotionId string, override konfidence.PromotionOverride) error {
	if override.SetBy == "" {
		return ErrActorMissing
	}
	if override.Reason == "" || len(override.Reason) > MaxOverrideReasonLength {
		return ErrReasonInvalid
	}

	vectorPromotion, err := r.Get(ctx, namespace, vectorPromotionId)
	if err != nil {
		return err
	}

	if existing := vectorPromotion.Status.Override; existing != nil {
		if existing.Type != override.Type {
			return overriddenError(existing)
		}
		return nil
	}
	if promotion.IsTerminal(vectorPromotion) {
		return finishedError(vectorPromotion)
	}
	if !promotiongate.GatesMutable(vectorPromotion) {
		return ErrPromotionExecuting
	}

	original := vectorPromotion.DeepCopy()
	override.SetAt = metav1.Now()
	vectorPromotion.Status.Override = &override
	patch := client.MergeFromWithOptions(original, client.MergeFromWithOptimisticLock{})
	if err := r.k8sClient.Status().Patch(ctx, vectorPromotion, patch); err != nil {
		switch {
		case apierrors.IsNotFound(err):
			return ErrVectorPromotionNotFound
		case apierrors.IsConflict(err):
			return ErrConflict
		default:
			return fmt.Errorf("failed to record %s override on VectorPromotion %q in namespace %q: %w",
				override.Type, vectorPromotionId, namespace, err)
		}
	}
	return nil
}

// mapGateSetError maps a promotiongate.Set failure to the repository's error
// vocabulary. action names the attempted operation for the fallback message.
func mapGateSetError(err error, key types.NamespacedName, action string) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, promotiongate.ErrNotFound):
		return ErrVectorPromotionNotFound
	case errors.Is(err, promotiongate.ErrGateNotDeclared):
		return ErrApprovalNotRequired
	case errors.Is(err, promotiongate.ErrPromotionNotMutable):
		return ErrPromotionExecuting
	case errors.Is(err, promotiongate.ErrConflict):
		return ErrConflict
	default:
		return fmt.Errorf("failed to %s VectorPromotion %q in namespace %q: %w", action, key.Name, key.Namespace, err)
	}
}

func finishedError(p *konfidence.VectorPromotion) error {
	cond := meta.FindStatusCondition(p.Status.Conditions, konfidence.ConditionTypeSucceeded)
	return fmt.Errorf("%w: %s: %s", ErrPromotionFinished, cond.Reason, cond.Message)
}

func overriddenError(override *konfidence.PromotionOverride) error {
	return fmt.Errorf("%w: %s by %s: %s", ErrPromotionOverridden, override.Type, override.SetBy, override.Reason)
}
