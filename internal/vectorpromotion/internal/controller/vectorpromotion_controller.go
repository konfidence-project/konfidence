package controller

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	EventActionUnknownPromotionStatus = "ReconcileRunningPromotion"
	EventActionStatusPatch            = "StatusPatch"
	EventActionReconciling            = "Reconciling"

	executionPendingMessage = "promotion execution is disabled pending the ADR-0032 execution rework " +
		"(konfidence-project#867)"
)

// VectorPromotionReconciler reconciles a VectorPromotion object.
type VectorPromotionReconciler struct {
	client.Client
	Recorder events.EventRecorder
}

// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectorpromotions,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectorpromotions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectorpromotionconfigs,verbs=get;list;watch

func (r *VectorPromotionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	ctx = logf.IntoContext(ctx, log)
	log.Info("reconciling vector promotion")

	vectorPromotion := &konfidence.VectorPromotion{}
	if err := r.Get(ctx, req.NamespacedName, vectorPromotion); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if meta.FindStatusCondition(vectorPromotion.Status.Conditions, konfidence.ConditionTypeSucceeded) != nil {
		return ctrl.Result{}, nil
	}

	// Take the snapshot before any modifications for the status patch.
	original := vectorPromotion.DeepCopy()

	log.Info(executionPendingMessage)
	r.Recorder.Eventf(vectorPromotion, nil, corev1.EventTypeNormal, "PromotionExecutionPending",
		EventActionReconciling, executionPendingMessage)
	if err := setAndPatchPromotionCondition(
		ctx, log, r.Client, r.Recorder, vectorPromotion, original,
		metav1.ConditionUnknown, konfidence.ReasonPromotionExecutionPending, executionPendingMessage); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func setAndPatchPromotionCondition(
	ctx context.Context,
	log logr.Logger,
	c client.Client,
	recorder events.EventRecorder,
	vectorPromotion, original *konfidence.VectorPromotion,
	status metav1.ConditionStatus,
	reason, message string) error {
	meta.SetStatusCondition(&vectorPromotion.Status.Conditions, metav1.Condition{
		Type:               konfidence.ConditionTypeSucceeded,
		Status:             status,
		ObservedGeneration: vectorPromotion.Generation,
		Reason:             reason,
		Message:            message,
	})
	vectorPromotion.Status.State = promotion.DeriveState(vectorPromotion)
	if !reflect.DeepEqual(vectorPromotion.Status, original.Status) {
		if err := c.Status().Patch(ctx, vectorPromotion, client.MergeFrom(original)); err != nil {
			log.Error(err, fmt.Sprintf("failed to patch promotion status of promotion %q in namespace %q",
				vectorPromotion.Name, vectorPromotion.Namespace))
			recorder.Eventf(vectorPromotion, nil, corev1.EventTypeWarning, "StatusPatchFailed", EventActionStatusPatch,
				fmt.Sprintf("wanted to set condition of type %q to status %q with reason %q and message %q "+
					"but failed with error: %s", konfidence.ConditionTypeSucceeded, status, reason, message, err.Error()))
			return fmt.Errorf("failed to patch VectorPromotion status: %w", err)
		}
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

// SetupWithManager sets up the controller with the Manager.
func (r *VectorPromotionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&konfidence.VectorPromotion{}, builder.WithPredicates(predicate.Funcs{
			CreateFunc:  func(e event.CreateEvent) bool { return true },
			UpdateFunc:  func(e event.UpdateEvent) bool { return false },
			DeleteFunc:  func(e event.DeleteEvent) bool { return false },
			GenericFunc: func(e event.GenericEvent) bool { return false },
		})).
		Named("vectorPromotion").
		Complete(r)
}
