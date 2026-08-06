package controller

import (
	"context"
	"fmt"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	deploymentClassNameField = "spec.deploymentClassName"
	deploymentClassNotFound  = "DeploymentClassNotFound"
)

// Reconciler marks targets whose class does not exist.
type Reconciler struct {
	client.Client
}

func NewReconciler(mgr manager.Manager) *Reconciler {
	return &Reconciler{Client: mgr.GetClient()}
}

// +kubebuilder:rbac:groups=konfidence.cloud,resources=deploymenttargets,verbs=get;list;watch
// +kubebuilder:rbac:groups=konfidence.cloud,resources=deploymenttargets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=konfidence.cloud,resources=deploymentclasses,verbs=get;list;watch

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	target := &konfidence.DeploymentTarget{}
	if err := r.Get(ctx, req.NamespacedName, target); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	class := &konfidence.DeploymentClass{}
	if err := r.Get(ctx, types.NamespacedName{Name: target.Spec.DeploymentClassName}, class); err == nil {
		return ctrl.Result{}, nil
	} else if !apierrors.IsNotFound(err) {
		return ctrl.Result{}, fmt.Errorf("failed to get DeploymentClass %q: %w", target.Spec.DeploymentClassName, err)
	}

	original := target.DeepCopy()
	meta.SetStatusCondition(&target.Status.Conditions, metav1.Condition{
		Type:               konfidence.DeploymentTargetReadyCondition,
		Status:             metav1.ConditionFalse,
		Reason:             deploymentClassNotFound,
		Message:            fmt.Sprintf("referenced DeploymentClass %q was not found, please install a deployer for this class", target.Spec.DeploymentClassName),
		ObservedGeneration: target.Generation,
	})
	if err := r.Status().Patch(ctx, target, client.MergeFrom(original)); err != nil {
		return ctrl.Result{}, fmt.Errorf("patch DeploymentTarget status: %w", err)
	}
	return ctrl.Result{}, nil
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &konfidence.DeploymentTarget{}, deploymentClassNameField,
		func(obj client.Object) []string {
			return []string{obj.(*konfidence.DeploymentTarget).Spec.DeploymentClassName}
		}); err != nil {
		return fmt.Errorf("index DeploymentTargets by deployment class: %w", err)
	}

	deletionsOnly := predicate.Funcs{
		CreateFunc:  func(event.CreateEvent) bool { return false },
		UpdateFunc:  func(event.UpdateEvent) bool { return false },
		DeleteFunc:  func(event.DeleteEvent) bool { return true },
		GenericFunc: func(event.GenericEvent) bool { return false },
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&konfidence.DeploymentTarget{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Watches(&konfidence.DeploymentClass{}, handler.EnqueueRequestsFromMapFunc(r.mapClassToTargets), builder.WithPredicates(deletionsOnly)).
		Named("deploymentTarget").
		Complete(r)
}

func (r *Reconciler) mapClassToTargets(ctx context.Context, obj client.Object) []reconcile.Request {
	targets := &konfidence.DeploymentTargetList{}
	if err := r.List(ctx, targets, client.MatchingFields{deploymentClassNameField: obj.GetName()}); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0, len(targets.Items))
	for i := range targets.Items {
		requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&targets.Items[i])})
	}
	return requests
}
