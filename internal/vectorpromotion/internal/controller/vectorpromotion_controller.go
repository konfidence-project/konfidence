package controller

import (
	"context"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcontroller "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/konfidence-project/konfidence/internal/vectorpromotion/internal/promotion"
)

const (
	VectorPromotionControllerName = "vector-promotion-controller"

	EventActionStatusPatch = "StatusPatch"
	EventActionGates       = "Gates"
	EventActionExecution   = "Execution"
	EventActionOverride    = "Override"

	// Watches provide prompt reconciliation; periodic observation also recovers
	// from a failed watch mapping or a sibling absent from a cached list.
	promotionResyncInterval = 30 * time.Second

	// promotionConfigNameField indexes promotions by their config name,
	// both in the manager cache (RegisterFieldIndexes) and server-side via the
	// CRD's selectableFields, so sibling listing works on cached and direct
	// clients alike.
	promotionConfigNameField = "spec.vectorPromotionConfigName"
)

// VectorPromotionReconciler executes cleared VectorPromotions: it waits for
// every declared gate to be satisfied (vectorpromotion_gates.go), finishes
// rejected promotions and records operator gate bypasses
// (vectorpromotion_override.go), serializes execution per VectorPromotionConfig
// (vectorpromotion_serialization.go), and writes the promoted vector to the
// target Stage (vectorpromotion_execution.go). See doc.go for the lifecycle
// narrative and the invariants that tie the phases together.
type VectorPromotionReconciler struct {
	client.Client
	Recorder events.EventRecorder
}

// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectorpromotions,verbs=get;list;watch
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
	if promotion.HasOverrideSet(vectorPromotion, konfidence.PromotionOverrideTypeReject) && vectorPromotion.Status.StartTime == nil {
		return ctrl.Result{}, r.reject(ctx, vectorPromotion)
	}
	if promotion.HasOverrideSet(vectorPromotion, konfidence.PromotionOverrideTypeBypassGates) {
		if err := r.markGatesBypassed(ctx, vectorPromotion); err != nil {
			return ctrl.Result{}, err
		}
	} else if !promotion.AllGatesSatisfied(vectorPromotion) {
		if err := r.reconcileGates(ctx, vectorPromotion); err != nil {
			return ctrl.Result{}, err
		}
	}
	result, err := r.reconcileExecution(ctx, vectorPromotion)
	if err == nil && !promotion.IsTerminal(vectorPromotion) && result.RequeueAfter == 0 {
		result.RequeueAfter = promotionResyncInterval
	}
	return result, err
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

// Events enqueue observations, never transitions. Promotion changes can affect
// older siblings' eligibility and unblock newer siblings; config changes can
// change outstanding limits. Both use the same config-scoped active set.
func (r *VectorPromotionReconciler) mapToActivePromotions(ctx context.Context, obj client.Object) []reconcile.Request {
	configName := obj.GetName()
	var excludeUID types.UID
	switch source := obj.(type) {
	case *konfidence.VectorPromotion:
		configName = source.Spec.VectorPromotionConfigName
		excludeUID = source.UID
	case *konfidence.VectorPromotionConfig:
	default:
		return nil
	}
	promotions, err := listPromotionsForConfig(ctx, r.Client, obj.GetNamespace(), configName)
	if err != nil {
		logf.FromContext(ctx).Error(err, "failed to map active promotions", "config", configName)
		return nil
	}
	var requests []reconcile.Request
	for i := range promotions {
		p := &promotions[i]
		if promotion.IsTerminal(p) || (excludeUID != "" && p.UID == excludeUID) {
			continue
		}
		requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(p)})
	}
	return requests
}

// Create/delete events affect the outstanding count. Updates fan out only when
// scheduling facts change, not for gate provenance or condition messages.
func promotionSchedulingChanged() predicate.Funcs {
	return predicate.Funcs{
		GenericFunc: func(e event.GenericEvent) bool { return false },
		UpdateFunc: func(e event.UpdateEvent) bool {
			old, okOld := e.ObjectOld.(*konfidence.VectorPromotion)
			current, okNew := e.ObjectNew.(*konfidence.VectorPromotion)
			return okOld && okNew && (supersedesOlderPromotions(old) != supersedesOlderPromotions(current) ||
				promotion.IsTerminal(old) != promotion.IsTerminal(current) ||
				promotion.IsRunning(old) != promotion.IsRunning(current))
		},
	}
}

// SetupWithManager sets up the controller with the Manager. Update events are
// admitted because gate changes and sibling terminations arrive as status updates.
func (r *VectorPromotionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&konfidence.VectorPromotion{}, builder.WithPredicates(predicate.Funcs{
			DeleteFunc:  func(e event.DeleteEvent) bool { return false },
			GenericFunc: func(e event.GenericEvent) bool { return false },
		})).
		Watches(&konfidence.VectorPromotion{},
			handler.EnqueueRequestsFromMapFunc(r.mapToActivePromotions),
			builder.WithPredicates(promotionSchedulingChanged())).
		Watches(&konfidence.VectorPromotionConfig{},
			handler.EnqueueRequestsFromMapFunc(r.mapToActivePromotions),
			builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		// Lifecycle decisions use one worker, with leader election across replicas.
		// A sequence watermark and optimistic Stage patch protect execution order
		// even when cached sibling observations lag behind a completed execution.
		WithOptions(ctrlcontroller.Options{MaxConcurrentReconciles: 1}).
		Named("vectorPromotion").
		Complete(r)
}
