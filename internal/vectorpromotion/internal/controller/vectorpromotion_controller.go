package controller

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	pkgctrl "github.com/konfidence-project/konfidence/pkg/controller"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

	// Retention defaults mirror the API defaults when the config omits a bound
	// or no longer exists.
	defaultKeepLastPromotions        = 10
	defaultKeepOutstandingPromotions = 10

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
// target Stage (vectorpromotion_execution.go), and deletes itself when it falls
// outside the terminal retention limit. See doc.go for the lifecycle
// narrative and the invariants that tie the phases together.
type VectorPromotionReconciler struct {
	client.Client
	Recorder events.EventRecorder
}

// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectorpromotions,verbs=get;list;watch;delete
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
		return r.reconcileTerminalRetention(ctx, vectorPromotion)
	}
	if promotion.HasOverrideSet(vectorPromotion, konfidence.PromotionOverrideTypeReject) && vectorPromotion.Status.StartTime == nil {
		return ctrl.Result{RequeueAfter: promotionResyncInterval}, r.reject(ctx, vectorPromotion)
	}
	if promotion.HasOverrideSet(vectorPromotion, konfidence.PromotionOverrideTypeBypassGates) {
		if err := r.markGatesBypassed(ctx, vectorPromotion); err != nil {
			return ctrl.Result{}, err
		}
	} else if !promotion.AllGatesSatisfied(vectorPromotion) {
		if err := r.reconcileGates(ctx, vectorPromotion); err != nil {
			return ctrl.Result{}, err
		}
		if promotion.AllGatesSatisfied(vectorPromotion) {
			// Readiness was established locally. Execute only after a later
			// reconciliation reads it from the shared promotion informer cache,
			// where sibling reconciliations can observe it too. The status watch
			// normally requeues us immediately; the timer is a recovery fallback.
			return ctrl.Result{RequeueAfter: promotionResyncInterval}, nil
		}
	}
	result, err := r.reconcileExecution(ctx, vectorPromotion)
	if err == nil && result.RequeueAfter == 0 {
		result.RequeueAfter = promotionResyncInterval
	}
	return result, err
}

// reconcileTerminalRetention never changes terminal status or deletes siblings.
// Every terminal outcome, including rejection, participates in the same count.
func (r *VectorPromotionReconciler) reconcileTerminalRetention(ctx context.Context, current *konfidence.VectorPromotion) (ctrl.Result, error) {
	keep := defaultKeepLastPromotions
	config, err := getPromotionConfig(ctx, r.Client, current)
	if err != nil && !apierrors.IsNotFound(err) {
		return ctrl.Result{}, fmt.Errorf("failed to fetch promotion configuration for retention: %w", err)
	}
	if err == nil && config.Spec.KeepLastPromotions != nil {
		keep = int(*config.Spec.KeepLastPromotions)
	}
	siblings, err := listSiblingPromotions(ctx, r.Client, current)
	if err != nil {
		return ctrl.Result{}, err
	}
	eligible := terminalPromotionsEligibleForRetention(append(siblings, *current))
	sort.Slice(eligible, func(i, j int) bool {
		if eligible[i].Spec.Sequence == eligible[j].Spec.Sequence {
			// Hand-crafted promotions may share a sequence. Stable ordering
			// prevents each sibling from deciding that the other should survive.
			return eligible[i].Name < eligible[j].Name
		}
		return promotion.Newer(&eligible[i], &eligible[j])
	})
	for i := range eligible {
		if eligible[i].UID == current.UID && i >= keep {
			return ctrl.Result{}, client.IgnoreNotFound(r.Delete(ctx, current, client.Preconditions{
				UID: &current.UID, ResourceVersion: &current.ResourceVersion,
			}))
		}
	}
	// Watches normally wake retained/deferred terminals promptly. Retry as a
	// fallback when a watch mapping failed or sibling observations were stale.
	return ctrl.Result{RequeueAfter: promotionResyncInterval}, nil
}

// Only terminal promotions strictly older than every non-terminal promotion
// participate in KeepLastPromotions. All others are reconsidered after a retry
// or a sibling/config event, regardless of their terminal outcome.
func terminalPromotionsEligibleForRetention(promotions []konfidence.VectorPromotion) []konfidence.VectorPromotion {
	var oldestActive *konfidence.VectorPromotion
	for i := range promotions {
		p := &promotions[i]
		if !promotion.IsTerminal(p) && (oldestActive == nil || promotion.Newer(oldestActive, p)) {
			oldestActive = p
		}
	}
	var eligible []konfidence.VectorPromotion
	for i := range promotions {
		p := &promotions[i]
		if promotion.IsTerminal(p) && (oldestActive == nil || promotion.Newer(oldestActive, p)) {
			eligible = append(eligible, *p)
		}
	}
	return eligible
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

// Promotion events resolve their parent config before using the same mapping
// as config events. The primary watch already queues the source promotion.
func (r *VectorPromotionReconciler) mapPromotionToSiblings(ctx context.Context, obj client.Object) []reconcile.Request {
	p, ok := obj.(*konfidence.VectorPromotion)
	if !ok {
		return nil
	}
	config, err := getPromotionConfig(ctx, r.Client, p)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			logf.FromContext(ctx).Error(err, "failed to fetch promotion config for promotion mapping", "promotion", client.ObjectKeyFromObject(p))
		}
		return nil
	}
	requests := r.mapConfigToPromotions(ctx, config)
	return slices.DeleteFunc(requests, func(request reconcile.Request) bool {
		return request.NamespacedName == client.ObjectKeyFromObject(p)
	})
}

// Events enqueue observations, never transitions. Both watches use this
// config-scoped set for supersession, execution, and both retention limits.
// Terminal siblings must also reconcile so they can delete themselves.
func (r *VectorPromotionReconciler) mapConfigToPromotions(ctx context.Context, obj client.Object) []reconcile.Request {
	config, ok := obj.(*konfidence.VectorPromotionConfig)
	if !ok {
		return nil
	}
	promotions, err := listPromotionsForConfig(ctx, r.Client, config.Namespace, config.Name)
	if err != nil {
		logf.FromContext(ctx).Error(err, "failed to map promotions", "config", client.ObjectKeyFromObject(config))
		return nil
	}
	var requests []reconcile.Request
	for i := range promotions {
		p := &promotions[i]
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

			if !okOld || !okNew {
				return false
			}

			// Older siblings reconsider whether they are superseded.
			supersessionChanged := supersedesOlderPromotions(old) != supersedesOlderPromotions(current)

			// Newer siblings reconsider whether they can execute.
			executionBlockingChanged := promotion.IsRunning(old) != promotion.IsRunning(current)

			// Waiting siblings reconsider the outstanding limit, including when
			// a rejected promotion finishes without changing supersession eligibility.
			// Terminal siblings also reconsider their own retention/deletion.
			terminalStateChanged := promotion.IsTerminal(old) != promotion.IsTerminal(current)

			return supersessionChanged || executionBlockingChanged || terminalStateChanged
		},
	}
}

// SetupWithManager sets up the controller with the Manager. Update events are
// admitted because gate changes and sibling terminations arrive as status updates.
func (r *VectorPromotionReconciler) SetupWithManager(mgr ctrl.Manager, excludedNamespaces ...pkgctrl.NamespaceExclusions) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&konfidence.VectorPromotion{}, builder.WithPredicates(predicate.Funcs{
			DeleteFunc:  func(e event.DeleteEvent) bool { return false },
			GenericFunc: func(e event.GenericEvent) bool { return false },
		})).
		Watches(&konfidence.VectorPromotion{},
			handler.EnqueueRequestsFromMapFunc(r.mapPromotionToSiblings),
			builder.WithPredicates(promotionSchedulingChanged())).
		Watches(&konfidence.VectorPromotionConfig{},
			handler.EnqueueRequestsFromMapFunc(r.mapConfigToPromotions),
			builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		// Lifecycle decisions use one worker, with leader election across replicas.
		// Persisted readiness/start transitions are observed through the shared
		// cache before the next step. Stage patches use optimistic locking.
		WithOptions(ctrlcontroller.Options{MaxConcurrentReconciles: 1}).
		Named("vectorPromotion").
		Complete(pkgctrl.WrapExcludedNamespaces(r, excludedNamespaces...))
}
