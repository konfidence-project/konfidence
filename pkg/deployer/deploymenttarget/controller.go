package deploymenttarget

import (
	"context"
	"fmt"

	konfidencev1alpha1 "github.com/konfidence-project/konfidence/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// ReasonInvalidParameters indicates malformed or invalid parameters.
	ReasonInvalidParameters = "InvalidParameters"
	// ReasonAccepted indicates that all validation checks passed and the DeploymentTarget is ready for use.
	ReasonAccepted = "Accepted"
	// ReasonValidationError indicates validation could not be completed.
	ReasonValidationError = "ValidationError"
	// ReasonInvalidValidationResult indicates the validator returned an unsupported result status.
	ReasonInvalidValidationResult = "InvalidValidationResult"
)

type reconciler struct {
	client                 client.Client
	controllerName         string
	supportedArtifactTypes konfidencev1alpha1.ArtifactTypeSet
	validateFunc           ValidationFunc
	recorder               events.EventRecorder
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

	result, validationErr := r.validate(ctx, dt)
	var reconcileErr error
	switch {
	case validationErr != nil:
		r.setReady(dt, metav1.ConditionFalse, ReasonValidationError, validationErr.Error())
		r.recordWarning(dt, ReasonValidationError, validationErr.Error())
		reconcileErr = validationErr
	case result != nil && result.Valid():
		message := result.Message()
		if message == "" {
			message = "parameters accepted"
		}
		r.setReady(dt, metav1.ConditionTrue, ReasonAccepted, message)
	case result != nil && !result.Valid():
		message := result.Message()
		if message == "" {
			message = "invalid parameters"
		}
		r.setReady(dt, metav1.ConditionFalse, ReasonInvalidParameters, message)
		r.recordWarning(dt, ReasonInvalidParameters, message)
	default:
		message := "validator returned a nil validation result"
		r.setReady(dt, metav1.ConditionFalse, ReasonInvalidValidationResult, message)
		reconcileErr = fmt.Errorf("%s", message)
	}

	patch := client.MergeFrom(original)
	if err := r.client.Status().Patch(ctx, dt, patch); err != nil {
		return ctrl.Result{}, fmt.Errorf("patch DeploymentTarget status: %w", err)
	}
	return ctrl.Result{}, reconcileErr
}

func (r *reconciler) validate(ctx context.Context, dt *konfidencev1alpha1.DeploymentTarget) (ValidationResult, error) {
	if r.validateFunc != nil {
		return r.validateFunc(ctx, r.client, dt)
	}

	return ValidationSuccess(""), nil
}

func (r *reconciler) recordWarning(target *konfidencev1alpha1.DeploymentTarget, reason, message string) {
	if r.recorder != nil {
		r.recorder.Eventf(target, nil, corev1.EventTypeWarning, reason, reason, "%s", message)
	}
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
