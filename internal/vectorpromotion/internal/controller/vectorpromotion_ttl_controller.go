package controller

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

// VectorPromotionTTLReconciler deletes terminal promotions by count and TTL.
// All lifecycle status transitions belong to the promotion's own reconciler.
type VectorPromotionTTLReconciler struct {
	client.Client
}

// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectorpromotions,verbs=get;list;watch;delete
// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectorpromotionconfigs,verbs=get;list;watch

func (r *VectorPromotionTTLReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	ctx = logf.IntoContext(ctx, log)

	vectorPromotion := &konfidence.VectorPromotion{}
	if err := r.Get(ctx, req.NamespacedName, vectorPromotion); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !promotion.IsTerminal(vectorPromotion) {
		return ctrl.Result{}, nil
	}

	promotions, err := listPromotionsForConfig(ctx, r.Client, vectorPromotion.Namespace, vectorPromotion.Spec.VectorPromotionConfigName)
	if err != nil {
		return ctrl.Result{}, err
	}
	if err := r.enforceTerminalRetention(ctx, vectorPromotion, promotions); err != nil {
		return ctrl.Result{}, err
	}
	if neededByOlderPromotion(vectorPromotion, promotions) {
		return ctrl.Result{RequeueAfter: siblingBlockedRequeueInterval}, nil
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

// enforceTerminalRetention deletes the oldest terminal promotions of the config
// beyond its keepLastPromotions bound, so short TTLs or missing TTLs cannot
// erase or grow the audit trail without limit.
func (r *VectorPromotionTTLReconciler) enforceTerminalRetention(
	ctx context.Context, vectorPromotion *konfidence.VectorPromotion, promotions []konfidence.VectorPromotion,
) error {
	keep, err := r.retentionBound(ctx, vectorPromotion)
	if err != nil {
		return err
	}

	terminal := make([]konfidence.VectorPromotion, 0, len(promotions))
	for _, candidate := range promotions {
		if promotion.IsTerminal(&candidate) {
			terminal = append(terminal, candidate)
		}
	}
	if len(terminal) <= keep {
		return nil
	}

	sort.Slice(terminal, func(i, j int) bool { return promotion.Newer(&terminal[i], &terminal[j]) })
	var errs []error
	for i := keep; i < len(terminal); i++ {
		if neededByOlderPromotion(&terminal[i], promotions) {
			continue
		}
		errs = append(errs, client.IgnoreNotFound(r.Delete(ctx, &terminal[i])))
	}
	return errors.Join(errs...)
}

// Retain supersession evidence until every older active promotion has observed
// it. Otherwise TTL/count cleanup could allow an old promotion to execute again
// after a restart or delayed watch event. Retention bounds are eventual.
func neededByOlderPromotion(candidate *konfidence.VectorPromotion, promotions []konfidence.VectorPromotion) bool {
	for i := range promotions {
		if !promotion.IsTerminal(&promotions[i]) && canSupersede(candidate, &promotions[i]) {
			return true
		}
	}
	return false
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

// mapConfigToPromotions enqueues one terminal promotion, when present, so
// configuration changes reapply finished-promotion retention.
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
	var terminal *konfidence.VectorPromotion
	for i := range promotions {
		candidate := &promotions[i]
		if promotion.IsTerminal(candidate) {
			if terminal == nil || promotion.Newer(candidate, terminal) {
				terminal = candidate
			}
		}
	}

	if terminal == nil {
		return nil
	}
	return []reconcile.Request{{NamespacedName: types.NamespacedName{
		Namespace: terminal.Namespace,
		Name:      terminal.Name,
	}}}
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
			DeleteFunc:  func(e event.DeleteEvent) bool { return false },
			GenericFunc: func(e event.GenericEvent) bool { return false },
		})).
		Watches(&konfidence.VectorPromotionConfig{},
			handler.EnqueueRequestsFromMapFunc(r.mapConfigToPromotions),
			builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Named("vectorPromotionTTL").
		Complete(r)
}
