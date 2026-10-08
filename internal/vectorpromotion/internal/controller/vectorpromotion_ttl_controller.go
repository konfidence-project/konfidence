package controller

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/konfidence-project/konfidence/internal/vectorpromotion/internal/promotion"
)

// Retention defaults mirror the kubebuilder defaults and apply when a
// promotion's config no longer exists.
const (
	defaultKeepLastPromotions        = 10
	defaultKeepOutstandingPromotions = 10
)

// VectorPromotionTTLReconciler enforces outstanding and finished promotion
// retention and deletes terminal promotions after their TTL expires.
type VectorPromotionTTLReconciler struct {
	client.Client
}

// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectorpromotions,verbs=get;list;watch;delete
// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectorpromotions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectorpromotionconfigs,verbs=get;list;watch

func (r *VectorPromotionTTLReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	ctx = logf.IntoContext(ctx, log)

	vectorPromotion := &konfidence.VectorPromotion{}
	if err := r.Get(ctx, req.NamespacedName, vectorPromotion); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !promotion.IsTerminal(vectorPromotion) {
		return ctrl.Result{}, r.enforceOutstandingRetention(ctx, vectorPromotion)
	}

	if err := r.enforceTerminalRetention(ctx, vectorPromotion); err != nil {
		return ctrl.Result{}, err
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

// enforceOutstandingRetention supersedes the oldest promotions that are still
// waiting at an approval gate when the configured bound is exceeded.
func (r *VectorPromotionTTLReconciler) enforceOutstandingRetention(
	ctx context.Context,
	vectorPromotion *konfidence.VectorPromotion,
) error {
	keep, err := r.outstandingRetentionBound(ctx, vectorPromotion)
	if err != nil {
		return err
	}

	siblings, err := listSiblingPromotions(ctx, r.Client, vectorPromotion)
	if err != nil {
		return err
	}
	outstanding := make([]konfidence.VectorPromotion, 0, len(siblings))
	for i := range siblings {
		if !promotion.IsTerminal(&siblings[i]) && !promotion.Cleared(&siblings[i]) {
			outstanding = append(outstanding, siblings[i])
		}
	}
	if len(outstanding) <= keep {
		return nil
	}

	// sorted from oldest to newest
	sort.SliceStable(outstanding, func(i, j int) bool {
		return promotion.Newer(&outstanding[j], &outstanding[i])
	})
	var errs []error
	for i := range outstanding[:len(outstanding)-keep] {
		candidate := &outstanding[i]
		original := candidate.DeepCopy()
		message := fmt.Sprintf(
			"promotion of vector %q was superseded because the limit of %d outstanding promotions was exceeded",
			candidate.Spec.Vector,
			keep,
		)
		setPromotionCondition(candidate, metav1.ConditionFalse,
			konfidence.ReasonPromotionOutstandingLimitReached, message)
		if err := patchPromotionStatus(ctx, r.Client, candidate, original); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// enforceTerminalRetention deletes the oldest terminal promotions of the config
// beyond its keepLastPromotions bound, so short TTLs or missing TTLs cannot
// erase or grow the audit trail without limit.
func (r *VectorPromotionTTLReconciler) enforceTerminalRetention(ctx context.Context, vectorPromotion *konfidence.VectorPromotion) error {
	keep, err := r.retentionBound(ctx, vectorPromotion)
	if err != nil {
		return err
	}

	siblings, err := listSiblingPromotions(ctx, r.Client, vectorPromotion)
	if err != nil {
		return err
	}
	terminal := make([]konfidence.VectorPromotion, 0, len(siblings))
	for _, sibling := range siblings {
		if promotion.IsTerminal(&sibling) {
			terminal = append(terminal, sibling)
		}
	}
	if len(terminal) <= keep {
		return nil
	}

	sort.Slice(terminal, func(i, j int) bool { return promotion.Newer(&terminal[i], &terminal[j]) })
	var errs []error
	for i := keep; i < len(terminal); i++ {
		errs = append(errs, client.IgnoreNotFound(r.Delete(ctx, &terminal[i])))
	}
	return errors.Join(errs...)
}

func (r *VectorPromotionTTLReconciler) retentionBound(ctx context.Context, vectorPromotion *konfidence.VectorPromotion) (int, error) {
	config, err := getPromotionConfig(ctx, r.Client, vectorPromotion)
	if apierrors.IsNotFound(err) {
		return defaultKeepLastPromotions, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to fetch promotion configuration for retention: %w", err)
	}
	if config.Spec.KeepLastPromotions == nil {
		return defaultKeepLastPromotions, nil
	}
	return int(*config.Spec.KeepLastPromotions), nil
}

func (r *VectorPromotionTTLReconciler) outstandingRetentionBound(
	ctx context.Context,
	vectorPromotion *konfidence.VectorPromotion,
) (int, error) {
	config, err := getPromotionConfig(ctx, r.Client, vectorPromotion)
	if apierrors.IsNotFound(err) {
		return defaultKeepOutstandingPromotions, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to fetch promotion configuration for outstanding retention: %w", err)
	}
	if config.Spec.KeepOutstandingPromotions == nil {
		return defaultKeepOutstandingPromotions, nil
	}
	return int(*config.Spec.KeepOutstandingPromotions), nil
}

// mapConfigToPromotions enqueues one outstanding and one terminal promotion,
// when present, so configuration changes reapply both retention bounds.
func (r *VectorPromotionTTLReconciler) mapConfigToPromotions(
	ctx context.Context,
	obj client.Object,
) []reconcile.Request {
	config, ok := obj.(*konfidence.VectorPromotionConfig)
	if !ok {
		return nil
	}
	promotions, err := listPromotionsForConfig(ctx, r.Client, config.Namespace, config.Name)
	if err != nil {
		logf.FromContext(ctx).Error(err, "failed to list promotions for retention config watch", "config", config.Name)
		return nil
	}
	var outstanding, terminal *konfidence.VectorPromotion
	for i := range promotions {
		candidate := &promotions[i]
		switch {
		case promotion.IsTerminal(candidate):
			if terminal == nil || promotion.Newer(candidate, terminal) {
				terminal = candidate
			}
		case !promotion.Cleared(candidate):
			if outstanding == nil || promotion.Newer(candidate, outstanding) {
				outstanding = candidate
			}
		}
	}

	requests := make([]reconcile.Request, 0, 2)
	for _, candidate := range []*konfidence.VectorPromotion{outstanding, terminal} {
		if candidate != nil {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
				Namespace: candidate.Namespace,
				Name:      candidate.Name,
			}})
		}
	}
	return requests
}

// NewVectorPromotionTTLReconciler wires a VectorPromotionTTLReconciler for the given manager.
func NewVectorPromotionTTLReconciler(mgr ctrl.Manager) *VectorPromotionTTLReconciler {
	return &VectorPromotionTTLReconciler{
		Client: mgr.GetClient(),
	}
}

// SetupWithManager sets up the promotion retention controller with the Manager.
// Promotion create and update events drive retention and TTL handling. Config
// generation changes reapply count-based retention with the new bounds.
func (r *VectorPromotionTTLReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&konfidence.VectorPromotion{}, builder.WithPredicates(predicate.Funcs{
			CreateFunc:  func(e event.CreateEvent) bool { return true },
			UpdateFunc:  func(e event.UpdateEvent) bool { return true },
			DeleteFunc:  func(e event.DeleteEvent) bool { return false },
			GenericFunc: func(e event.GenericEvent) bool { return false },
		})).
		Watches(&konfidence.VectorPromotionConfig{},
			handler.EnqueueRequestsFromMapFunc(r.mapConfigToPromotions),
			builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Named("vectorPromotionTTL").
		Complete(r)
}
