package controller

import (
	"context"
	"fmt"
	"time"

	global "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	mcbuilder "sigs.k8s.io/multicluster-runtime/pkg/builder"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
	mcreconcile "sigs.k8s.io/multicluster-runtime/pkg/reconcile"

	"github.com/konfidence-project/konfidence/internal/galaxy/vector-promotion/internal/controller/domain"
)

// VectorPromotionTTLReconciler deletes VectorPromotion objects that have exceeded their TTL
// after reaching a terminal phase.
type VectorPromotionTTLReconciler struct {
	Mgr    mcmanager.Manager
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=galaxy.konfidence.cloud,resources=vectorpromotions,verbs=get;list;watch;delete

func (r *VectorPromotionTTLReconciler) Reconcile(ctx context.Context, req mcreconcile.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx).WithValues("cluster", req.ClusterName)
	ctx = logf.IntoContext(ctx, log)

	cluster, err := r.Mgr.GetCluster(ctx, req.ClusterName)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get cluster: %w", err)
	}
	clusterClient := cluster.GetClient()

	vectorPromotion := &global.VectorPromotion{}
	if err := clusterClient.Get(ctx, req.NamespacedName, vectorPromotion); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	shouldDelete, remaining := domain.TTLStatus(vectorPromotion)
	if remaining > 0 {
		log.Info("VectorPromotion TTL not yet expired, requeueing", "remaining", remaining.Round(time.Second))
		return ctrl.Result{RequeueAfter: remaining}, nil
	}
	if !shouldDelete {
		return ctrl.Result{}, nil
	}

	log.Info("VectorPromotion TTL expired, deleting")
	if err := clusterClient.Delete(ctx, vectorPromotion); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the TTL controller with the Manager.
// Only Create and Update events are processed — Delete and Generic events are irrelevant.
// Update events allow the controller to react when a TTL is added to an existing object
// or when the Promoted condition is set.
func (r *VectorPromotionTTLReconciler) SetupWithManager(mgr mcmanager.Manager) error {
	return mcbuilder.ControllerManagedBy(mgr).
		For(&global.VectorPromotion{}, mcbuilder.WithPredicates(predicate.Funcs{
			CreateFunc:  func(e event.CreateEvent) bool { return true },
			UpdateFunc:  func(e event.UpdateEvent) bool { return true },
			DeleteFunc:  func(e event.DeleteEvent) bool { return false },
			GenericFunc: func(e event.GenericEvent) bool { return false },
		})).
		Named("vectorPromotionTTL").
		Complete(r)
}
