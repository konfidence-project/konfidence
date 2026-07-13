package controller

import (
	"context"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/konfidence-project/konfidence/internal/vectorpromotion/internal/promotion"
)

// VectorPromotionTTLReconciler deletes VectorPromotion objects that have exceeded their TTL
// after reaching a terminal phase.
type VectorPromotionTTLReconciler struct {
	client.Client
}

// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectorpromotions,verbs=get;list;watch;delete

func (r *VectorPromotionTTLReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	ctx = logf.IntoContext(ctx, log)

	vectorPromotion := &konfidence.VectorPromotion{}
	if err := r.Get(ctx, req.NamespacedName, vectorPromotion); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	shouldDelete, remaining := promotion.TTLStatus(vectorPromotion)
	if remaining > 0 {
		log.Info("VectorPromotion TTL not yet expired, requeueing", "remaining", remaining.Round(time.Second))
		return ctrl.Result{RequeueAfter: remaining}, nil
	}
	if !shouldDelete {
		return ctrl.Result{}, nil
	}

	log.Info("VectorPromotion TTL expired, deleting")
	if err := r.Delete(ctx, vectorPromotion); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	return ctrl.Result{}, nil
}

// NewVectorPromotionTTLReconciler wires a VectorPromotionTTLReconciler for the given manager.
func NewVectorPromotionTTLReconciler(mgr ctrl.Manager) *VectorPromotionTTLReconciler {
	return &VectorPromotionTTLReconciler{
		Client: mgr.GetClient(),
	}
}

// SetupWithManager sets up the TTL controller with the Manager.
// Only Create and Update events are processed — Delete and Generic events are irrelevant.
// Update events allow the controller to react when a TTL is added to an existing object
// or when the Promoted condition is set.
func (r *VectorPromotionTTLReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&konfidence.VectorPromotion{}, builder.WithPredicates(predicate.Funcs{
			CreateFunc:  func(e event.CreateEvent) bool { return true },
			UpdateFunc:  func(e event.UpdateEvent) bool { return true },
			DeleteFunc:  func(e event.DeleteEvent) bool { return false },
			GenericFunc: func(e event.GenericEvent) bool { return false },
		})).
		Named("vectorPromotionTTL").
		Complete(r)
}
