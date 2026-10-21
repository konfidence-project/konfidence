package deploymenttarget

import (
	"context"
	"fmt"

	konfidencev1alpha1 "github.com/konfidence-project/konfidence/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// ReasonInvalidParameters indicates malformed or invalid parameters.
	ReasonInvalidParameters = "InvalidParameters"
	// ReasonAccepted indicates that all validation checks passed and the DeploymentTarget is ready for use.
	ReasonAccepted = "Accepted"
)

type reconciler struct {
	client                 client.Client
	controllerName         string
	supportedArtifactTypes konfidencev1alpha1.ArtifactTypeSet
	watchers               []watch
	validate               ValidationFunc
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	dt := &konfidencev1alpha1.DeploymentTarget{}
	if err := r.client.Get(ctx, req.NamespacedName, dt); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !dt.IsSupported(r.supportedArtifactTypes) {
		return ctrl.Result{}, nil
	}

	log.Info("reconciling DeploymentTarget", "artifactType", dt.Spec.ArtifactType)
	if err := r.setController(ctx, dt); err != nil {
		return ctrl.Result{}, err
	}

	if r.validate != nil {
		if err := r.validate(ctx, r.client, dt); err != nil {
			if err := r.setReady(ctx, dt, metav1.ConditionFalse, ReasonInvalidParameters, fmt.Sprintf("invalid parameters: %v", err)); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
	}

	if err := r.setReady(ctx, dt, metav1.ConditionTrue, ReasonAccepted, "parameters accepted"); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *reconciler) setController(ctx context.Context, target *konfidencev1alpha1.DeploymentTarget) error {
	if target.Status.Controller == r.controllerName {
		return nil
	}

	patch := client.MergeFrom(target.DeepCopy())
	target.Status.Controller = r.controllerName
	if err := r.client.Status().Patch(ctx, target, patch); err != nil {
		return fmt.Errorf("patch DeploymentTarget controller: %w", err)
	}
	return nil
}

func (r *reconciler) setReady(ctx context.Context, target *konfidencev1alpha1.DeploymentTarget, status metav1.ConditionStatus, reason, message string) error {
	patch := client.MergeFrom(target.DeepCopy())
	meta.SetStatusCondition(&target.Status.Conditions, metav1.Condition{
		Type:               konfidencev1alpha1.DeploymentTargetReadyCondition,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: target.Generation,
		LastTransitionTime: metav1.Now(),
	})
	if err := r.client.Status().Patch(ctx, target, patch); err != nil {
		return fmt.Errorf("patch DeploymentTarget status: %w", err)
	}
	return nil
}
