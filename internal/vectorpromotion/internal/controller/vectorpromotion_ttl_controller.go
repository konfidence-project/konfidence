package controller

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/konfidence-project/konfidence/internal/vectorpromotion/internal/promotion"
)

// defaultKeepLastPromotions mirrors the kubebuilder default on
// `VectorPromotionConfig.spec.keepLastPromotions` for promotions whose config
// no longer exists.
const defaultKeepLastPromotions = 10

// VectorPromotionTTLReconciler deletes VectorPromotion objects that have exceeded their TTL
// after reaching a terminal phase.
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

	// Retention and TTL only ever act on terminal promotions.
	if !promotion.IsTerminal(vectorPromotion) {
		return ctrl.Result{}, nil
	}

	if err := r.enforceRetention(ctx, vectorPromotion); err != nil {
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

// enforceRetention deletes the oldest terminal promotions of the config
// beyond its keepLastPromotions bound, so short TTLs or missing TTLs cannot
// erase or grow the audit trail without limit.
func (r *VectorPromotionTTLReconciler) enforceRetention(ctx context.Context, vectorPromotion *konfidence.VectorPromotion) error {
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
