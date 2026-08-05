package controller

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/konfidence-project/konfidence/internal/vectorpromotion/internal/promotion"
)

const (
	VectorPromotionControllerName = "vector-promotion-controller"

	EventActionStatusPatch = "StatusPatch"
	EventActionApproval    = "Approval"
	EventActionExecution   = "Execution"

	// siblingBlockedRequeueInterval paces re-evaluation while another
	// promotion of the same config is executing.
	siblingBlockedRequeueInterval = 10 * time.Second
)

// VectorPromotionReconciler executes approved VectorPromotions: it gates on
// approval, serializes execution per VectorPromotionConfig, supersedes stale
// approved promotions, and writes the promoted vector to the target Stage.
type VectorPromotionReconciler struct {
	client.Client
	Recorder events.EventRecorder
}

// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectorpromotions,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectorpromotions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectorpromotionconfigs,verbs=get;list;watch
// +kubebuilder:rbac:groups=konfidence.cloud,resources=landscapes,verbs=get;list;watch
// +kubebuilder:rbac:groups=konfidence.cloud,resources=stages,verbs=get;list;watch;update;patch

func (r *VectorPromotionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	ctx = logf.IntoContext(ctx, log)

	vectorPromotion := &konfidence.VectorPromotion{}
	if err := r.Get(ctx, req.NamespacedName, vectorPromotion); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if promotion.IsTerminal(vectorPromotion) {
		return ctrl.Result{}, nil
	}

	if !promotion.IsApproved(vectorPromotion) {
		return ctrl.Result{}, r.reconcileApproval(ctx, vectorPromotion)
	}

	return r.reconcileExecution(ctx, vectorPromotion)
}

// reconcileApproval moves an unapproved promotion into WaitingForApproval or
// approves it directly when it does not require approval. The status update
// retriggers reconciliation, which then proceeds to execution.
func (r *VectorPromotionReconciler) reconcileApproval(ctx context.Context, vectorPromotion *konfidence.VectorPromotion) error {
	original := vectorPromotion.DeepCopy()

	if vectorPromotion.Spec.RequireApproval {
		setApprovedCondition(vectorPromotion, metav1.ConditionFalse, konfidence.ReasonPromotionWaitingForApproval,
			"promotion requires approval before execution")
		if err := r.patchStatusWithEvent(ctx, vectorPromotion, original); err != nil {
			return err
		}
		r.Recorder.Eventf(vectorPromotion, nil, corev1.EventTypeNormal, "PromotionWaitingForApproval",
			EventActionApproval, "promotion is waiting for approval")
		return nil
	}

	setApprovedCondition(vectorPromotion, metav1.ConditionTrue, konfidence.ReasonPromotionAutoApproved,
		"promotion is approved automatically because it does not require approval")
	if err := r.patchStatusWithEvent(ctx, vectorPromotion, original); err != nil {
		return err
	}
	r.Recorder.Eventf(vectorPromotion, nil, corev1.EventTypeNormal, "PromotionAutoApproved",
		EventActionApproval, "promotion approved automatically")
	return nil
}

// reconcileExecution serializes execution per config: at most one promotion is
// in progress, only the newest approved one executes, and stale approved
// promotions are superseded.
func (r *VectorPromotionReconciler) reconcileExecution(ctx context.Context, vectorPromotion *konfidence.VectorPromotion) (ctrl.Result, error) {
	siblings, err := r.listSiblings(ctx, vectorPromotion)
	if err != nil {
		return ctrl.Result{}, err
	}

	for i := range siblings {
		if siblings[i].UID != vectorPromotion.UID && promotion.IsInProgress(&siblings[i]) {
			return ctrl.Result{RequeueAfter: siblingBlockedRequeueInterval}, nil
		}
	}

	newest := promotion.NewestApproved(siblings)
	if newest == nil || newest.UID != vectorPromotion.UID {
		return ctrl.Result{}, r.supersede(ctx, vectorPromotion, newest)
	}

	if err := r.supersedeStaleSiblings(ctx, vectorPromotion, siblings); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, r.execute(ctx, vectorPromotion)
}

// listSiblings returns all promotions in the promotion's namespace that
// reference the same VectorPromotionConfig, including the promotion itself.
func (r *VectorPromotionReconciler) listSiblings(ctx context.Context, vectorPromotion *konfidence.VectorPromotion) ([]konfidence.VectorPromotion, error) {
	list := &konfidence.VectorPromotionList{}
	if err := r.List(ctx, list, client.InNamespace(vectorPromotion.Namespace)); err != nil {
		return nil, fmt.Errorf("failed to list sibling promotions: %w", err)
	}
	siblings := make([]konfidence.VectorPromotion, 0, len(list.Items))
	for _, item := range list.Items {
		if item.Spec.VectorPromotionConfigRef == vectorPromotion.Spec.VectorPromotionConfigRef {
			siblings = append(siblings, item)
		}
	}
	return siblings, nil
}

// supersedeStaleSiblings marks every other non-terminal sibling as superseded
// before the newest promotion executes.
func (r *VectorPromotionReconciler) supersedeStaleSiblings(ctx context.Context, newest *konfidence.VectorPromotion, siblings []konfidence.VectorPromotion) error {
	var errs []error
	for i := range siblings {
		if siblings[i].UID == newest.UID || promotion.IsTerminal(&siblings[i]) {
			continue
		}
		errs = append(errs, r.supersede(ctx, &siblings[i], newest))
	}
	return errors.Join(errs...)
}

// supersede terminates a promotion because a newer approved promotion for the
// same config exists.
func (r *VectorPromotionReconciler) supersede(ctx context.Context, vectorPromotion, newest *konfidence.VectorPromotion) error {
	message := "promotion was superseded by a newer approved promotion"
	if newest != nil {
		message = fmt.Sprintf("promotion was superseded by newer approved promotion %q", newest.Name)
	}
	original := vectorPromotion.DeepCopy()
	setPromotionCondition(vectorPromotion, metav1.ConditionFalse, konfidence.ReasonPromotionSuperseded, message)
	if err := r.patchStatusWithEvent(ctx, vectorPromotion, original); err != nil {
		return err
	}
	r.Recorder.Eventf(vectorPromotion, nil, corev1.EventTypeNormal, "PromotionSuperseded", EventActionExecution, message)
	return nil
}

// execute writes the pinned vector to the target Stage and marks the
// promotion succeeded. A missing config is terminal; an unresolved landscape
// or stage is transient and retried with backoff.
func (r *VectorPromotionReconciler) execute(ctx context.Context, vectorPromotion *konfidence.VectorPromotion) error {
	original := vectorPromotion.DeepCopy()

	config, err := getPromotionConfig(ctx, r.Client, vectorPromotion)
	if apierrors.IsNotFound(err) {
		msg := fmt.Sprintf("promotion configuration %q not found", vectorPromotion.Spec.VectorPromotionConfigRef)
		setPromotionCondition(vectorPromotion, metav1.ConditionFalse, konfidence.ReasonPromotionConfigurationNotFound, msg)
		return r.patchStatusWithEvent(ctx, vectorPromotion, original)
	}
	if err != nil {
		return fmt.Errorf("failed to fetch promotion configuration: %w", err)
	}

	if !promotion.IsInProgress(vectorPromotion) {
		setPromotionCondition(vectorPromotion, metav1.ConditionFalse, konfidence.ReasonPromotionRunning, "promotion is executing")
		if err := r.patchStatusWithEvent(ctx, vectorPromotion, original); err != nil {
			return err
		}
		original = vectorPromotion.DeepCopy()
	}

	stage, err := r.resolveTargetStage(ctx, config)
	if err != nil {
		r.Recorder.Eventf(vectorPromotion, nil, corev1.EventTypeWarning, "PromotionTargetUnresolved",
			EventActionExecution, err.Error())
		return err
	}

	if err := r.promoteStage(ctx, stage, vectorPromotion.Spec.Vector); err != nil {
		return err
	}

	msg := fmt.Sprintf("promoted vector to stage %q in namespace %q", stage.Name, stage.Namespace)
	setPromotionCondition(vectorPromotion, metav1.ConditionTrue, konfidence.ReasonPromotionSucceeded, msg)
	if err := r.patchStatusWithEvent(ctx, vectorPromotion, original); err != nil {
		return err
	}
	r.Recorder.Eventf(vectorPromotion, nil, corev1.EventTypeNormal, "PromotionSuccessful", EventActionExecution, msg)
	return nil
}

// resolveTargetStage resolves the config's target Stage through the Landscape
// in the config's namespace.
func (r *VectorPromotionReconciler) resolveTargetStage(ctx context.Context, config *konfidence.VectorPromotionConfig) (*konfidence.Stage, error) {
	landscape := &konfidence.Landscape{}
	key := types.NamespacedName{Namespace: config.Namespace, Name: config.Spec.Target.Landscape}
	if err := r.Get(ctx, key, landscape); err != nil {
		return nil, fmt.Errorf("failed to resolve landscape %q: %w", config.Spec.Target.Landscape, err)
	}
	if landscape.Status.Namespace == "" {
		return nil, fmt.Errorf("landscape %q has no managed namespace yet", landscape.Name)
	}

	stage := &konfidence.Stage{}
	key = types.NamespacedName{Namespace: landscape.Status.Namespace, Name: config.Spec.Target.Name}
	if err := r.Get(ctx, key, stage); err != nil {
		return nil, fmt.Errorf("failed to resolve stage %q in landscape namespace %q: %w",
			config.Spec.Target.Name, landscape.Status.Namespace, err)
	}
	return stage, nil
}

// promoteStage writes the promoted vector to the stage spec if it differs.
func (r *VectorPromotionReconciler) promoteStage(ctx context.Context, stage *konfidence.Stage, vector string) error {
	if stage.Spec.Vector == vector {
		return nil
	}
	original := stage.DeepCopy()
	stage.Spec.Vector = vector
	if err := r.Patch(ctx, stage, client.MergeFrom(original)); err != nil {
		return fmt.Errorf("failed to patch vector onto stage %q in namespace %q: %w", stage.Name, stage.Namespace, err)
	}
	return nil
}

// patchStatusWithEvent patches the promotion status if it changed and emits a
// warning event when the patch fails.
func (r *VectorPromotionReconciler) patchStatusWithEvent(ctx context.Context, vectorPromotion, original *konfidence.VectorPromotion) error {
	err := patchPromotionStatus(ctx, r.Client, vectorPromotion, original)
	if err != nil {
		r.Recorder.Eventf(vectorPromotion, nil, corev1.EventTypeWarning, "StatusPatchFailed", EventActionStatusPatch, err.Error())
	}
	return err
}

// setPromotionCondition writes the Succeeded condition and refreshes the
// derived status.state.
func setPromotionCondition(
	vectorPromotion *konfidence.VectorPromotion,
	status metav1.ConditionStatus,
	reason, message string,
) {
	meta.SetStatusCondition(&vectorPromotion.Status.Conditions, metav1.Condition{
		Type:               konfidence.ConditionTypeSucceeded,
		Status:             status,
		ObservedGeneration: vectorPromotion.Generation,
		Reason:             reason,
		Message:            message,
	})
	vectorPromotion.Status.State = promotion.DeriveState(vectorPromotion)
}

// setApprovedCondition writes the Approved condition and refreshes the
// derived status.state.
func setApprovedCondition(
	vectorPromotion *konfidence.VectorPromotion,
	status metav1.ConditionStatus,
	reason, message string,
) {
	meta.SetStatusCondition(&vectorPromotion.Status.Conditions, metav1.Condition{
		Type:               konfidence.ConditionTypeApproved,
		Status:             status,
		ObservedGeneration: vectorPromotion.Generation,
		Reason:             reason,
		Message:            message,
	})
	vectorPromotion.Status.State = promotion.DeriveState(vectorPromotion)
}

// patchPromotionStatus patches the promotion status if it changed relative to original.
func patchPromotionStatus(
	ctx context.Context,
	c client.Client,
	vectorPromotion, original *konfidence.VectorPromotion,
) error {
	if reflect.DeepEqual(vectorPromotion.Status, original.Status) {
		return nil
	}
	if err := c.Status().Patch(ctx, vectorPromotion, client.MergeFrom(original)); err != nil {
		return fmt.Errorf("failed to patch status of VectorPromotion %q in namespace %q: %w",
			vectorPromotion.Name, vectorPromotion.Namespace, err)
	}
	return nil
}

// NewVectorPromotionReconciler wires a VectorPromotionReconciler for the given manager.
func NewVectorPromotionReconciler(mgr ctrl.Manager) *VectorPromotionReconciler {
	return &VectorPromotionReconciler{
		Client:   mgr.GetClient(),
		Recorder: mgr.GetEventRecorder(VectorPromotionControllerName),
	}
}

// SetupWithManager sets up the controller with the Manager. Update events are
// admitted because approvals and sibling terminations arrive as status updates.
func (r *VectorPromotionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&konfidence.VectorPromotion{}, builder.WithPredicates(predicate.Funcs{
			CreateFunc:  func(e event.CreateEvent) bool { return true },
			UpdateFunc:  func(e event.UpdateEvent) bool { return true },
			DeleteFunc:  func(e event.DeleteEvent) bool { return false },
			GenericFunc: func(e event.GenericEvent) bool { return false },
		})).
		Named("vectorPromotion").
		Complete(r)
}
