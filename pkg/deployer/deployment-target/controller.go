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
	validateFunc           ValidationFunc
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	dt := &konfidencev1alpha1.DeploymentTarget{}
	err := r.client.Get(ctx, req.NamespacedName, dt)
	if err != nil || !r.supportedArtifactTypes.Contains(dt.ArtifactType()) {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("reconciling DeploymentTarget", "artifactType", dt.Spec.ArtifactType)
	original := dt.DeepCopy()

	dt.Status.Controller = r.controllerName

	err = r.validate(ctx, dt)
	if err != nil {
		r.setReady(dt, metav1.ConditionFalse, ReasonInvalidParameters, fmt.Sprintf("invalid parameters: %v", err))
	} else {
		r.setReady(dt, metav1.ConditionTrue, ReasonAccepted, "parameters accepted")
	}

	patch := client.MergeFrom(original)
	if err := r.client.Status().Patch(ctx, dt, patch); err != nil {
		return ctrl.Result{}, fmt.Errorf("patch DeploymentTarget status: %w", err)
	}
	return ctrl.Result{}, nil
}

func (r *reconciler) validate(ctx context.Context, dt *konfidencev1alpha1.DeploymentTarget) error {
	if r.validateFunc != nil {
		return r.validateFunc(ctx, r.client, dt)
	}

	return nil
}

func (r *reconciler) setReady(target *konfidencev1alpha1.DeploymentTarget, status metav1.ConditionStatus, reason, message string) {
	meta.SetStatusCondition(&target.Status.Conditions, metav1.Condition{
		Type:               konfidencev1alpha1.DeploymentTargetReadyCondition,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: target.Generation,
		LastTransitionTime: metav1.Now(),
	})
}
