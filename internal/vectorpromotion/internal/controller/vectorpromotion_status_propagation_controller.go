package controller

import (
	"context"
	"fmt"
	"reflect"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/vectorpromotion/internal/promotion"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// VectorPromotionStatusPropagationReconciler ensures that the status condition of the VectorPromotion are propagated to the
// respective VectorPromotionConfig.
type VectorPromotionStatusPropagationReconciler struct {
	client.Client
}

const statusPropagationReconcileInterval = time.Second * 1

// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectorpromotions,verbs=get;list;watch
// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectorpromotions/status,verbs=get
// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectorpromotionconfigs,verbs=get;list;watch
// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectorpromotionconfigs/status,verbs=get;update;patch

func (r *VectorPromotionStatusPropagationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	ctx = logf.IntoContext(ctx, log)

	p := &konfidence.VectorPromotion{}
	if err := r.Get(ctx, req.NamespacedName, p); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if promotion.IsPending(p) {
		return ctrl.Result{RequeueAfter: statusPropagationReconcileInterval}, nil
	}

	config, err := getPromotionConfig(ctx, r.Client, p)
	if apierrors.IsNotFound(err) {
		return ctrl.Result{RequeueAfter: statusPropagationReconcileInterval}, nil
	}
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get VectorPromotionConfig: %w", err)
	}

	// Skip when the config already captured strictly newer conditions, or the
	// exact same ones. Timestamp comparison alone is not enough: condition
	// timestamps have second resolution and a reason-only transition (e.g.
	// Running to Superseded) can share its predecessor's timestamp.
	promotionCondition := meta.FindStatusCondition(p.Status.Conditions, konfidence.ConditionTypeSucceeded)
	configCondition := meta.FindStatusCondition(config.Status.LastPromotionConditions, konfidence.ConditionTypeSucceeded)
	if configCondition != nil && promotionCondition.LastTransitionTime.Before(&configCondition.LastTransitionTime) {
		return requeueIfNotTerminal(p), nil
	}
	if reflect.DeepEqual(config.Status.LastPromotionConditions, p.Status.Conditions) {
		return requeueIfNotTerminal(p), nil
	}

	if err := patchPromotionConfigStatus(ctx, r.Client, p, config); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to patch VectorPromotionConfig status: %w", err)
	}

	return requeueIfNotTerminal(p), nil
}

func requeueIfNotTerminal(p *konfidence.VectorPromotion) ctrl.Result {
	if promotion.IsTerminal(p) {
		return ctrl.Result{}
	}
	return ctrl.Result{RequeueAfter: statusPropagationReconcileInterval}
}

func patchPromotionConfigStatus(
	ctx context.Context, c client.Client,
	p *konfidence.VectorPromotion, config *konfidence.VectorPromotionConfig,
) error {
	originalConfig := config.DeepCopy()
	config.Status.LastPromotionConditions = p.Status.Conditions
	if promotion.IsSucceeded(p) {
		config.Status.LastSuccessfulPromotionConditions = p.Status.Conditions
	}
	if err := c.Status().Patch(ctx, config, client.MergeFrom(originalConfig)); err != nil {
		return err
	}
	return nil
}

// NewVectorPromotionStatusPropagationReconciler wires a VectorPromotionStatusPropagationReconciler for the given manager.
func NewVectorPromotionStatusPropagationReconciler(mgr ctrl.Manager) *VectorPromotionStatusPropagationReconciler {
	return &VectorPromotionStatusPropagationReconciler{
		Client: mgr.GetClient(),
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *VectorPromotionStatusPropagationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&konfidence.VectorPromotion{}, builder.WithPredicates(predicate.Funcs{
			CreateFunc:  func(e event.CreateEvent) bool { return true },
			UpdateFunc:  func(e event.UpdateEvent) bool { return false },
			DeleteFunc:  func(e event.DeleteEvent) bool { return false },
			GenericFunc: func(e event.GenericEvent) bool { return false },
		})).
		Named("vectorPromotionStatusPropagation").
		Complete(r)
}
