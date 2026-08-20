package v1alpha1

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var deploymentclasslog = logf.Log.WithName("deploymentclass-webhook")

// +kubebuilder:rbac:groups=konfidence.cloud,resources=deploymentclasses,verbs=get;list;watch

// SetupDeploymentClassWebhookWithManager registers the webhook with the manager.
func SetupDeploymentClassWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &DeploymentClass{}).
		WithValidator(&DeploymentClassValidator{Client: mgr.GetClient()}).
		Complete()
}

//nolint:lll // Kubebuilder annotations are intentionally long.
// +kubebuilder:webhook:path=/validate-konfidence-cloud-v1alpha1-deploymentclass,mutating=false,failurePolicy=fail,sideEffects=None,groups=konfidence.cloud,resources=deploymentclasses,verbs=create;update,versions=v1alpha1,name=vdeploymentclass.konfidence.cloud,admissionReviewVersions=v1

// DeploymentClassValidator validates DeploymentClass resources.
// +kubebuilder:object:generate=false
type DeploymentClassValidator struct {
	Client client.Client
}

// ValidateCreate validates a DeploymentClass on creation.
func (v *DeploymentClassValidator) ValidateCreate(ctx context.Context, obj *DeploymentClass) (admission.Warnings, error) {
	deploymentclasslog.Info("validating DeploymentClass creation", "name", obj.Name, "type", obj.Spec.Type)

	if err := v.validateTypeUniqueness(ctx, obj, nil); err != nil {
		return nil, err
	}

	return nil, nil
}

// ValidateUpdate validates a DeploymentClass on update.
func (v *DeploymentClassValidator) ValidateUpdate(ctx context.Context, oldObj, newObj *DeploymentClass) (admission.Warnings, error) {
	deploymentclasslog.Info("validating DeploymentClass update", "name", newObj.Name, "type", newObj.Spec.Type)

	// If type hasn't changed, no validation needed
	if oldObj.Spec.Type == newObj.Spec.Type {
		return nil, nil
	}

	if err := v.validateTypeUniqueness(ctx, newObj, oldObj); err != nil {
		return nil, err
	}

	return nil, nil
}

// ValidateDelete validates a DeploymentClass on deletion (always allow).
func (v *DeploymentClassValidator) ValidateDelete(ctx context.Context, obj *DeploymentClass) (admission.Warnings, error) {
	return nil, nil
}

// validateTypeUniqueness checks that the spec.type is unique across all DeploymentClasses.
func (v *DeploymentClassValidator) validateTypeUniqueness(ctx context.Context, obj *DeploymentClass, _ *DeploymentClass) error {
	// List all DeploymentClasses
	list := &DeploymentClassList{}
	if err := v.Client.List(ctx, list); err != nil {
		return apierrors.NewInternalError(fmt.Errorf("failed to list DeploymentClasses: %w", err))
	}

	// Check for duplicates
	for i := range list.Items {
		existing := &list.Items[i]

		// Skip self
		if existing.Name == obj.Name {
			continue
		}

		// Check if type matches
		if existing.Spec.Type == obj.Spec.Type {
			return field.Invalid(
				field.NewPath("spec").Child("type"),
				obj.Spec.Type,
				fmt.Sprintf("type must be unique across all DeploymentClasses (already used by DeploymentClass %q)", existing.Name),
			)
		}
	}

	return nil
}
