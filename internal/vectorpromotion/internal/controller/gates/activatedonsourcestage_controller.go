package gates

import (
	"context"
	"errors"
	"fmt"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/vectorpromotion/internal/promotion"
	"github.com/konfidence-project/konfidence/pkg/promotiongate"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ActivatedOnSourceStageControllerName = "activated-on-source-stage-controller"

	ReasonActivatedOnSourceStage             = "ActivatedOnSource"
	ReasonSourceStageUnavailable             = "SourceStageUnavailable"
	ReasonWaitingForSourceStageActivation    = "WaitingForSourceStageActivation"
	activatedOnSourceStageSetBy              = "konfidence/activated-on-source-stage-controller"
	promotionSourceStageField                = "activatedOnSourceStage.sourceStage"
	activatedOnSourceStageLandscapeNamespace = "activatedOnSourceStage.landscapeNamespace"
	pendingGateRequeueAfter                  = time.Minute
)

// ActivatedOnSourceStageReconciler owns the core gate that requires a vector
// to have completed activation on the promotion's source Stage.
type ActivatedOnSourceStageReconciler struct {
	client.Client
}

// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectorpromotions,verbs=get;list;watch
// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectorpromotions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=konfidence.cloud,resources=landscapes,verbs=get;list;watch
// +kubebuilder:rbac:groups=konfidence.cloud,resources=stages,verbs=get;list;watch
// +kubebuilder:rbac:groups=konfidence.cloud,resources=stageversions,verbs=get;list;watch

// RegisterFieldIndexes registers indexes used to map a ready StageVersion to
// waiting promotions without scanning every promotion in the cluster.
func RegisterFieldIndexes(ctx context.Context, mgr ctrl.Manager) error {
	indexer := mgr.GetFieldIndexer()
	if err := indexer.IndexField(ctx, &konfidence.VectorPromotion{}, promotionSourceStageField,
		func(obj client.Object) []string {
			vectorPromotion, ok := obj.(*konfidence.VectorPromotion)
			if !ok || vectorPromotion.Spec.Source.Kind != konfidence.StageKind ||
				!promotiongate.Declared(vectorPromotion, konfidence.GateTypeActivatedOnSourceStage) {
				return nil
			}
			return []string{sourceStageKey(vectorPromotion.Spec.Source.Landscape, vectorPromotion.Spec.Source.Name)}
		}); err != nil {
		return err
	}
	return indexer.IndexField(ctx, &konfidence.Landscape{}, activatedOnSourceStageLandscapeNamespace,
		func(obj client.Object) []string {
			landscape, ok := obj.(*konfidence.Landscape)
			if !ok || landscape.Status.Namespace == "" {
				return nil
			}
			return []string{landscape.Status.Namespace}
		})
}

func NewActivatedOnSourceStageReconciler(mgr ctrl.Manager) *ActivatedOnSourceStageReconciler {
	return &ActivatedOnSourceStageReconciler{Client: mgr.GetClient()}
}

func (r *ActivatedOnSourceStageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	vectorPromotion := &konfidence.VectorPromotion{}
	if err := r.Get(ctx, req.NamespacedName, vectorPromotion); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if !promotiongate.Declared(vectorPromotion, konfidence.GateTypeActivatedOnSourceStage) ||
		promotiongate.StateOf(vectorPromotion, konfidence.GateTypeActivatedOnSourceStage) == konfidence.PromotionGateStateSatisfied ||
		!promotiongate.GatesMutable(vectorPromotion) {
		return ctrl.Result{}, nil
	}

	update, err := r.evaluateGate(ctx, vectorPromotion)
	if err != nil {
		return ctrl.Result{}, err
	}
	err = promotiongate.Set(ctx, r.Client, req.NamespacedName, update)
	if errors.Is(err, promotiongate.ErrPromotionNotMutable) || errors.Is(err, promotiongate.ErrNotFound) {
		return ctrl.Result{}, nil
	}
	if err != nil {
		return ctrl.Result{}, err
	}
	if update.State == konfidence.PromotionGateStatePending {
		return ctrl.Result{RequeueAfter: pendingGateRequeueAfter}, nil
	}
	return ctrl.Result{}, nil
}

func (r *ActivatedOnSourceStageReconciler) evaluateGate(
	ctx context.Context,
	vectorPromotion *konfidence.VectorPromotion,
) (promotiongate.Update, error) {
	update := promotiongate.Update{
		Type:   konfidence.GateTypeActivatedOnSourceStage,
		State:  konfidence.PromotionGateStatePending,
		Reason: ReasonWaitingForSourceStageActivation,
		SetBy:  activatedOnSourceStageSetBy,
	}

	source := vectorPromotion.Spec.Source
	if source.Kind != konfidence.StageKind || source.Landscape == "" || source.Name == "" {
		update.State = konfidence.PromotionGateStateError
		update.Reason = ReasonSourceStageUnavailable
		update.Message = "The ActivatedOnSourceStage gate requires a Stage source"
		return update, nil
	}

	landscape := &konfidence.Landscape{}
	landscapeKey := types.NamespacedName{Namespace: vectorPromotion.Namespace, Name: source.Landscape}
	if err := r.Get(ctx, landscapeKey, landscape); err != nil {
		if !apierrors.IsNotFound(err) {
			return promotiongate.Update{}, fmt.Errorf("read source landscape %q: %w", landscapeKey, err)
		}
		update.State = konfidence.PromotionGateStateError
		update.Reason = ReasonSourceStageUnavailable
		update.Message = fmt.Sprintf("Source landscape %q is not available", landscapeKey)
		return update, nil
	}
	if landscape.Status.Namespace == "" {
		update.State = konfidence.PromotionGateStateError
		update.Reason = ReasonSourceStageUnavailable
		update.Message = fmt.Sprintf("Source landscape %q has no managed namespace", landscape.Name)
		return update, nil
	}

	stageKey := types.NamespacedName{Namespace: landscape.Status.Namespace, Name: source.Name}
	if err := r.Get(ctx, stageKey, &konfidence.Stage{}); err != nil {
		if !apierrors.IsNotFound(err) {
			return promotiongate.Update{}, fmt.Errorf("read source stage %q: %w", stageKey, err)
		}
		update.State = konfidence.PromotionGateStateError
		update.Reason = ReasonSourceStageUnavailable
		update.Message = fmt.Sprintf("Source stage %q is not available", stageKey)
		return update, nil
	}

	stageVersions := &konfidence.StageVersionList{}
	if err := r.List(ctx, stageVersions, client.InNamespace(landscape.Status.Namespace)); err != nil {
		return promotiongate.Update{}, fmt.Errorf("list source StageVersions in namespace %q: %w", landscape.Status.Namespace, err)
	}
	for i := range stageVersions.Items {
		stageVersion := &stageVersions.Items[i]
		if stageVersion.Spec.StageRef != nil && stageVersion.Spec.StageRef.Name == source.Name &&
			stageVersion.Spec.Vector == vectorPromotion.Spec.Vector &&
			meta.IsStatusConditionTrue(stageVersion.Status.Conditions, konfidence.StageVersionReady) {
			update.State = konfidence.PromotionGateStateSatisfied
			update.Reason = ReasonActivatedOnSourceStage
			update.Message = "Vector activated on source stage"
			return update, nil
		}
	}
	return update, nil
}

func (r *ActivatedOnSourceStageReconciler) mapStageVersionToPromotions(
	ctx context.Context,
	obj client.Object,
) []reconcile.Request {
	stageVersion, ok := obj.(*konfidence.StageVersion)
	if !ok || stageVersion.Spec.StageRef == nil {
		return nil
	}

	landscapes := &konfidence.LandscapeList{}
	if err := r.List(ctx, landscapes, client.MatchingFields{
		activatedOnSourceStageLandscapeNamespace: stageVersion.Namespace,
	}); err != nil {
		logf.FromContext(ctx).Error(err, "failed to resolve landscape for StageVersion", "stageVersion", client.ObjectKeyFromObject(stageVersion))
		return nil
	}

	var requests []reconcile.Request
	for i := range landscapes.Items {
		landscape := &landscapes.Items[i]
		promotions := &konfidence.VectorPromotionList{}
		if err := r.List(ctx, promotions,
			client.InNamespace(landscape.Namespace),
			client.MatchingFields{promotionSourceStageField: sourceStageKey(landscape.Name, stageVersion.Spec.StageRef.Name)},
		); err != nil {
			logf.FromContext(ctx).Error(err, "failed to list promotions for StageVersion", "stageVersion", client.ObjectKeyFromObject(stageVersion))
			continue
		}
		for j := range promotions.Items {
			vectorPromotion := &promotions.Items[j]
			if vectorPromotion.Spec.Vector != stageVersion.Spec.Vector || !promotion.IsWaiting(vectorPromotion) {
				continue
			}
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(vectorPromotion)})
		}
	}
	return requests
}

func sourceStageKey(landscape, stage string) string {
	return landscape + "/" + stage
}

func hasActivatedOnSourceStageGate() predicate.Funcs {
	return predicate.NewPredicateFuncs(func(obj client.Object) bool {
		vectorPromotion, ok := obj.(*konfidence.VectorPromotion)
		return ok && promotiongate.Declared(vectorPromotion, konfidence.GateTypeActivatedOnSourceStage)
	})
}

func stageVersionBecameReady() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			stageVersion, ok := e.Object.(*konfidence.StageVersion)
			return ok && meta.IsStatusConditionTrue(stageVersion.Status.Conditions, konfidence.StageVersionReady)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldStageVersion, oldOK := e.ObjectOld.(*konfidence.StageVersion)
			newStageVersion, newOK := e.ObjectNew.(*konfidence.StageVersion)
			return oldOK && newOK &&
				!meta.IsStatusConditionTrue(oldStageVersion.Status.Conditions, konfidence.StageVersionReady) &&
				meta.IsStatusConditionTrue(newStageVersion.Status.Conditions, konfidence.StageVersionReady)
		},
		DeleteFunc:  func(event.DeleteEvent) bool { return false },
		GenericFunc: func(event.GenericEvent) bool { return false },
	}
}

func (r *ActivatedOnSourceStageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&konfidence.VectorPromotion{}, builder.WithPredicates(hasActivatedOnSourceStageGate())).
		Watches(
			&konfidence.StageVersion{},
			handler.EnqueueRequestsFromMapFunc(r.mapStageVersionToPromotions),
			builder.WithPredicates(stageVersionBecameReady()),
		).
		Named(ActivatedOnSourceStageControllerName).
		Complete(r)
}
