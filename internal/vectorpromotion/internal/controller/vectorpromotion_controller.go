package controller

import (
	"context"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcontroller "sigs.k8s.io/controller-runtime/pkg/controller"
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

	// promotionConfigRefField indexes promotions by their config reference,
	// both in the manager cache (RegisterFieldIndexes) and server-side via the
	// CRD's selectableFields, so sibling listing works on cached and direct
	// clients alike.
	promotionConfigRefField = "spec.vectorPromotionConfigRef"
)

// VectorPromotionReconciler executes approved VectorPromotions: it gates on
// approval (vectorpromotion_approval.go), serializes execution per
// VectorPromotionConfig (vectorpromotion_serialization.go), and writes the
// promoted vector to the target Stage (vectorpromotion_execution.go). See
// doc.go for the lifecycle narrative and the invariants that tie the phases
// together.
type VectorPromotionReconciler struct {
	client.Client
	Recorder events.EventRecorder
}

// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectorpromotions,verbs=get;list;watch
// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectorpromotions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectorpromotionconfigs,verbs=get;list;watch
// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectorpromotionconfigs/status,verbs=get;update;patch
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

// RegisterFieldIndexes registers the cache indexes the promotion controllers
// rely on. Call once per manager, before registering the controllers.
func RegisterFieldIndexes(ctx context.Context, mgr ctrl.Manager) error {
	return mgr.GetFieldIndexer().IndexField(ctx, &konfidence.VectorPromotion{}, promotionConfigRefField,
		func(obj client.Object) []string {
			vectorPromotion, ok := obj.(*konfidence.VectorPromotion)
			if !ok {
				return nil
			}
			return []string{vectorPromotion.Spec.VectorPromotionConfigRef}
		})
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
		// The serialization invariants (one InProgress per config, newest
		// approved wins) are enforced by read-check-write over the informer
		// cache and depend on a single writer: exactly one worker here, and
		// leader election in multi-replica deployments. The unlocked config
		// status patches in vectorpromotion_execution.go depend on the same
		// single-writer property.
		WithOptions(ctrlcontroller.Options{MaxConcurrentReconciles: 1}).
		Named("vectorPromotion").
		Complete(r)
}
