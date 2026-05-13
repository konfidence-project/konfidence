package controller

import (
	"context"
	"fmt"
	"time"

	global "github.com/konfidence-project/crds/api/global/v1alpha1"
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

	"github.com/konfidence-project/gcp-vector-promotion-controller/internal/controller/domain"
)

// VectorPromotionStatusPropagationReconciler enures that the status condition of the VectorPromotion are propagated to the
// respective VectorPromotionConfig.
type VectorPromotionStatusPropagationReconciler struct {
	Mgr    mcmanager.Manager
	Scheme *runtime.Scheme
}

const statusPropagationReconcileInterval = time.Second * 1

// +kubebuilder:rbac:groups=global.konfidence.cloud,resources=vectorpromotions,verbs=get;list;watch
// +kubebuilder:rbac:groups=global.konfidence.cloud,resources=vectorpromotions/status,verbs=get
// +kubebuilder:rbac:groups=global.konfidence.cloud,resources=vectorpromotionconfigs,verbs=get;list;watch
// +kubebuilder:rbac:groups=global.konfidence.cloud,resources=vectorpromotionconfigs/status,verbs=get;update;patch

func (r *VectorPromotionStatusPropagationReconciler) Reconcile(ctx context.Context, req mcreconcile.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx).WithValues("cluster", req.ClusterName)
	ctx = logf.IntoContext(ctx, log)

	cluster, err := r.Mgr.GetCluster(ctx, req.ClusterName)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get cluster: %w", err)
	}
	clusterClient := cluster.GetClient()

	promotion := &global.VectorPromotion{}
	if err := clusterClient.Get(ctx, req.NamespacedName, promotion); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if domain.IsPending(promotion) {
		return ctrl.Result{RequeueAfter: statusPropagationReconcileInterval}, nil
	}

	config, err := getPromotionConfig(ctx, clusterClient, promotion)
	if apierrors.IsNotFound(err) {
		return ctrl.Result{RequeueAfter: statusPropagationReconcileInterval}, nil
	}
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get VectorPromotionConfig: %w", err)
	}

	promotionCondition := meta.FindStatusCondition(promotion.Status.Conditions, global.ConditionTypeSucceeded)
	configCondition := meta.FindStatusCondition(config.Status.LastPromotionConditions, global.ConditionTypeSucceeded)
	if configCondition != nil && !promotionCondition.LastTransitionTime.After(configCondition.LastTransitionTime.Time) {
		return requeueIfNotTerminal(promotion), nil
	}

	if err := patchPromotionConfigStatus(ctx, clusterClient, promotion, config); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to patch VectorPromotionConfig status: %w", err)
	}

	return requeueIfNotTerminal(promotion), nil
}

func requeueIfNotTerminal(promotion *global.VectorPromotion) ctrl.Result {
	if domain.IsTerminal(promotion) {
		return ctrl.Result{}
	}
	return ctrl.Result{RequeueAfter: statusPropagationReconcileInterval}
}

func patchPromotionConfigStatus(ctx context.Context, clusterClient client.Client, promotion *global.VectorPromotion, config *global.VectorPromotionConfig) error {
	originalConfig := config.DeepCopy()
	config.Status.LastPromotionConditions = promotion.Status.Conditions
	if domain.IsSucceeded(promotion) {
		config.Status.LastSuccessfulPromotionConditions = promotion.Status.Conditions
	}
	if err := clusterClient.Status().Patch(ctx, config, client.MergeFrom(originalConfig)); err != nil {
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VectorPromotionStatusPropagationReconciler) SetupWithManager(mgr mcmanager.Manager) error {
	return mcbuilder.ControllerManagedBy(mgr).
		For(&global.VectorPromotion{}, mcbuilder.WithPredicates(predicate.Funcs{
			CreateFunc:  func(e event.CreateEvent) bool { return true },
			UpdateFunc:  func(e event.UpdateEvent) bool { return false },
			DeleteFunc:  func(e event.DeleteEvent) bool { return false },
			GenericFunc: func(e event.GenericEvent) bool { return false },
		})).
		Named("vectorPromotionStatusPropagation").
		Complete(r)
}
