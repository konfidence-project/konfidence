package target

import (
	"context"
	"fmt"

	konfidencev1alpha1 "github.com/konfidence-project/konfidence/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// DeploymentTargetReasonInvalidParameters indicates malformed or invalid parameters.
	DeploymentTargetReasonInvalidParameters = "InvalidParameters"
	// DeploymentTargetReasonAccepted indicates that all validation checks passed and the DeploymentTarget is ready for use.
	DeploymentTargetReasonAccepted = "Accepted"
)

type ValidationFunc func(ctx context.Context, c client.Reader, target *konfidencev1alpha1.DeploymentTarget) error

type Watcher struct {
	Object       client.Object
	EventHandler handler.TypedEventHandler[client.Object, reconcile.Request]
	Opts         []builder.WatchesOption
}

// DeploymentTargetReconciler reconciles supported DeploymentTarget resources.
type DeploymentTargetReconciler struct {
	client.Client

	ControllerName         string
	SupportedArtifactTypes konfidencev1alpha1.ArtifactTypes
	Watchers               []Watcher
	Validate               ValidationFunc
}

// +kubebuilder:rbac:groups=konfidence.cloud,resources=deploymenttargets,verbs=get;list;watch
// +kubebuilder:rbac:groups=konfidence.cloud,resources=deploymenttargets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *DeploymentTargetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	dt := &konfidencev1alpha1.DeploymentTarget{}
	if err := r.Get(ctx, req.NamespacedName, dt); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !dt.IsSupported(r.SupportedArtifactTypes) {
		return ctrl.Result{}, nil
	}

	log.Info("reconciling DeploymentTarget", "artifactType", dt.Spec.ArtifactType)
	if err := r.setController(ctx, dt); err != nil {
		return ctrl.Result{}, err
	}

	if r.Validate != nil {
		if err := r.Validate(ctx, r.Client, dt); err != nil {
			if err := r.setReady(ctx, dt, metav1.ConditionFalse, DeploymentTargetReasonInvalidParameters, fmt.Sprintf("invalid parameters: %v", err)); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
	}

	if err := r.setReady(ctx, dt, metav1.ConditionTrue, DeploymentTargetReasonAccepted, "parameters accepted"); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *DeploymentTargetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	supportedTarget := predicate.NewPredicateFuncs(func(obj client.Object) bool {
		target, ok := obj.(*konfidencev1alpha1.DeploymentTarget)
		if !ok {
			return false
		}
		return target.IsSupported(r.SupportedArtifactTypes)
	})

	cb := builder.TypedControllerManagedBy[reconcile.Request](mgr).
		For(&konfidencev1alpha1.DeploymentTarget{}, builder.WithPredicates(predicate.GenerationChangedPredicate{}, supportedTarget))

	for _, watch := range r.Watchers {
		cb = cb.Watches(watch.Object, watch.EventHandler, watch.Opts...)
	}

	return cb.Named(r.ControllerName).Complete(r)
}

func (r *DeploymentTargetReconciler) setController(ctx context.Context, target *konfidencev1alpha1.DeploymentTarget) error {
	if target.Status.Controller == r.ControllerName {
		return nil
	}

	patch := client.MergeFrom(target.DeepCopy())
	target.Status.Controller = r.ControllerName
	if err := r.Status().Patch(ctx, target, patch); err != nil {
		return fmt.Errorf("patch DeploymentTarget controller: %w", err)
	}
	return nil
}

func (r *DeploymentTargetReconciler) setReady(ctx context.Context, target *konfidencev1alpha1.DeploymentTarget, status metav1.ConditionStatus, reason, message string) error {
	patch := client.MergeFrom(target.DeepCopy())
	meta.SetStatusCondition(&target.Status.Conditions, metav1.Condition{
		Type:               konfidencev1alpha1.DeploymentTargetReadyCondition,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: target.Generation,
		LastTransitionTime: metav1.Now(),
	})
	if err := r.Status().Patch(ctx, target, patch); err != nil {
		return fmt.Errorf("patch DeploymentTarget status: %w", err)
	}
	return nil
}
