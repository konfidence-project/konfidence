package controller

import (
	"context"
	"fmt"
	"time"

	galaxy "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/galaxy/vectorpromotion/internal/promotion"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	mcbuilder "sigs.k8s.io/multicluster-runtime/pkg/builder"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
	mcreconcile "sigs.k8s.io/multicluster-runtime/pkg/reconcile"
)

// VectorPromotionStatusPropagationReconciler enures that the status condition of the VectorPromotion are propagated to the
// respective VectorPromotionConfig.
type VectorPromotionStatusPropagationReconciler struct {
	Mgr    mcmanager.Manager
	Scheme *runtime.Scheme
}

const statusPropagationReconcileInterval = time.Second * 1

// +kubebuilder:rbac:groups=galaxy.konfidence.cloud,resources=vectorpromotions,verbs=get;list;watch
// +kubebuilder:rbac:groups=galaxy.konfidence.cloud,resources=vectorpromotions/status,verbs=get
// +kubebuilder:rbac:groups=galaxy.konfidence.cloud,resources=vectorpromotionconfigs,verbs=get;list;watch
// +kubebuilder:rbac:groups=galaxy.konfidence.cloud,resources=vectorpromotionconfigs/status,verbs=get;update;patch

func (r *VectorPromotionStatusPropagationReconciler) Reconcile(ctx context.Context, req mcreconcile.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx).WithValues("cluster", req.ClusterName)
	ctx = logf.IntoContext(ctx, log)

	cluster, err := r.Mgr.GetCluster(ctx, req.ClusterName)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get cluster: %w", err)
	}
	clusterClient := cluster.GetClient()

	p := &galaxy.VectorPromotion{}
	if err := clusterClient.Get(ctx, req.NamespacedName, p); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if promotion.IsPending(p) {
		return ctrl.Result{RequeueAfter: statusPropagationReconcileInterval}, nil
	}

	config, err := getPromotionConfig(ctx, clusterClient, p)
	if apierrors.IsNotFound(err) {
		return ctrl.Result{RequeueAfter: statusPropagationReconcileInterval}, nil
	}
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get VectorPromotionConfig: %w", err)
	}

	promotionCondition := meta.FindStatusCondition(p.Status.Conditions, galaxy.ConditionTypeSucceeded)
	configCondition := meta.FindStatusCondition(config.Status.LastPromotionConditions, galaxy.ConditionTypeSucceeded)
	if configCondition != nil && !promotionCondition.LastTransitionTime.After(configCondition.LastTransitionTime.Time) {
		return requeueIfNotTerminal(p), nil
	}

	if err := patchPromotionConfigStatus(ctx, clusterClient, p, config); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to patch VectorPromotionConfig status: %w", err)
	}

	return requeueIfNotTerminal(p), nil
}

func requeueIfNotTerminal(p *galaxy.VectorPromotion) ctrl.Result {
	if promotion.IsTerminal(p) {
		return ctrl.Result{}
	}
	return ctrl.Result{RequeueAfter: statusPropagationReconcileInterval}
}

func patchPromotionConfigStatus(
	ctx context.Context, clusterClient client.Client,
	p *galaxy.VectorPromotion, config *galaxy.VectorPromotionConfig,
) error {
	originalConfig := config.DeepCopy()
	config.Status.LastPromotionConditions = p.Status.Conditions
	if promotion.IsSucceeded(p) {
		config.Status.LastSuccessfulPromotionConditions = p.Status.Conditions
	}
	if err := clusterClient.Status().Patch(ctx, config, client.MergeFrom(originalConfig)); err != nil {
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VectorPromotionStatusPropagationReconciler) SetupWithManager(mgr mcmanager.Manager) error {
	return mcbuilder.ControllerManagedBy(mgr).
		For(&galaxy.VectorPromotion{}, mcbuilder.WithPredicates(predicate.Funcs{
			CreateFunc:  func(e event.CreateEvent) bool { return true },
			UpdateFunc:  func(e event.UpdateEvent) bool { return false },
			DeleteFunc:  func(e event.DeleteEvent) bool { return false },
			GenericFunc: func(e event.GenericEvent) bool { return false },
		})).
		Named("vectorPromotionStatusPropagation").
		Complete(r)
}
