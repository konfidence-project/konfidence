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
	EventActionGates       = "Gates"
	EventActionExecution   = "Execution"

	// promotionConfigNameField indexes promotions by their config name,
	// both in the manager cache (RegisterFieldIndexes) and server-side via the
	// CRD's selectableFields, so sibling listing works on cached and direct
	// clients alike.
	promotionConfigNameField = "spec.vectorPromotionConfigName"
)

// VectorPromotionReconciler executes cleared VectorPromotions: it waits for
// every declared gate to be satisfied (vectorpromotion_gates.go), serializes execution per
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
	if !promotion.AllGatesSatisfied(vectorPromotion) {
		if err := r.reconcileGates(ctx, vectorPromotion); err != nil {
			return ctrl.Result{}, err
		}
		// Check if gates are now satisfied after reconciliation
		if !promotion.AllGatesSatisfied(vectorPromotion) {
			return ctrl.Result{}, nil
		}
	}
	return r.reconcileExecution(ctx, vectorPromotion)
}

// RegisterFieldIndexes registers the cache indexes the promotion controllers
// rely on. Call once per manager, before registering the controllers.
func RegisterFieldIndexes(ctx context.Context, mgr ctrl.Manager) error {
	indexer := mgr.GetFieldIndexer()
	if err := indexer.IndexField(ctx, &konfidence.VectorPromotion{}, promotionConfigNameField,
		func(obj client.Object) []string {
			vectorPromotion, ok := obj.(*konfidence.VectorPromotion)
			if !ok {
				return nil
			}
			return []string{vectorPromotion.Spec.VectorPromotionConfigName}
		}); err != nil {
		return err
	}
	if err := indexer.IndexField(ctx, &konfidence.VectorPromotionConfig{}, configSourceTemplateField,
		func(obj client.Object) []string {
			config, ok := obj.(*konfidence.VectorPromotionConfig)
			if !ok || config.Spec.Source.Kind != konfidence.VectorTemplateKind {
				return nil
			}
			return []string{config.Spec.Source.Name}
		}); err != nil {
		return err
	}
	if err := indexer.IndexField(ctx, &konfidence.VectorPromotionConfig{}, configSourceStageField,
		func(obj client.Object) []string {
			config, ok := obj.(*konfidence.VectorPromotionConfig)
			if !ok || config.Spec.Source.Kind != konfidence.StageKind {
				return nil
			}
			return []string{config.Spec.Source.Landscape + "/" + config.Spec.Source.Name}
		}); err != nil {
		return err
	}
	if err := indexer.IndexField(ctx, &konfidence.VectorPromotionConfig{}, configTargetStageField,
		func(obj client.Object) []string {
			config, ok := obj.(*konfidence.VectorPromotionConfig)
			if !ok {
				return nil
			}
			return []string{config.Spec.Target.Landscape + "/" + config.Spec.Target.Name}
		}); err != nil {
		return err
	}
	return indexer.IndexField(ctx, &konfidence.Landscape{}, landscapeNamespaceField,
		func(obj client.Object) []string {
			landscape, ok := obj.(*konfidence.Landscape)
			if !ok || landscape.Status.Namespace == "" {
				return nil
			}
			return []string{landscape.Status.Namespace}
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
// admitted because gate changes and sibling terminations arrive as status updates.
func (r *VectorPromotionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&konfidence.VectorPromotion{}, builder.WithPredicates(predicate.Funcs{
			CreateFunc:  func(e event.CreateEvent) bool { return true },
			UpdateFunc:  func(e event.UpdateEvent) bool { return true },
			DeleteFunc:  func(e event.DeleteEvent) bool { return false },
			GenericFunc: func(e event.GenericEvent) bool { return false },
		})).
		// The serialization invariants (one InProgress per config, newest
		// cleared wins) are enforced by read-check-write over the informer
		// cache and depend on a single writer: exactly one worker here, and
		// leader election in multi-replica deployments.
		WithOptions(ctrlcontroller.Options{MaxConcurrentReconciles: 1}).
		Named("vectorPromotion").
		Complete(r)
}
