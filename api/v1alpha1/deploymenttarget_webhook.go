package v1alpha1

import (
	"context"
	"fmt"

	pkgwebhook "github.com/konfidence-project/konfidence/pkg/webhook"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var deploymenttargetlog = logf.Log.WithName("deploymenttarget-webhook")

// +kubebuilder:rbac:groups=konfidence.cloud,resources=deploymenttargets,verbs=get;list;watch

// SetupDeploymentTargetWebhookWithManager registers the webhook with the manager.
func SetupDeploymentTargetWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &DeploymentTarget{}).
		WithValidator(&DeploymentTargetValidator{Client: mgr.GetClient()}).
		Complete()
}

//nolint:lll // Kubebuilder annotations are intentionally long.
// +kubebuilder:webhook:path=/validate-konfidence-cloud-v1alpha1-deploymenttarget,mutating=false,failurePolicy=fail,sideEffects=None,groups=konfidence.cloud,resources=deploymenttargets,verbs=create;update,versions=v1alpha1,name=vdeploymenttarget.konfidence.cloud,admissionReviewVersions=v1

// DeploymentTargetValidator validates DeploymentTarget resources.
// +kubebuilder:object:generate=false
type DeploymentTargetValidator struct {
	Client client.Client
}

// ValidateCreate validates a DeploymentTarget on creation.
func (v *DeploymentTargetValidator) ValidateCreate(ctx context.Context, obj *DeploymentTarget) (admission.Warnings, error) {
	deploymenttargetlog.Info("validating DeploymentTarget creation",
		"name", obj.Name,
		"namespace", obj.Namespace,
		"deploymentClassName", obj.Spec.DeploymentClassName,
	)
	if err := pkgwebhook.ValidateLandscapeNamespace(ctx, v.Client, obj.Namespace); err != nil {
		return nil, err
	}
	return nil, v.validateDeploymentClassUniqueness(ctx, obj)
}

// ValidateUpdate validates a DeploymentTarget on update.
func (v *DeploymentTargetValidator) ValidateUpdate(ctx context.Context, _ *DeploymentTarget, newObj *DeploymentTarget) (admission.Warnings, error) {
	deploymenttargetlog.Info("validating DeploymentTarget update",
		"name", newObj.Name,
		"namespace", newObj.Namespace,
		"deploymentClassName", newObj.Spec.DeploymentClassName,
	)
	return nil, pkgwebhook.ValidateLandscapeNamespace(ctx, v.Client, newObj.Namespace)
}

// ValidateDelete validates a DeploymentTarget on deletion.
func (v *DeploymentTargetValidator) ValidateDelete(_ context.Context, _ *DeploymentTarget) (admission.Warnings, error) {
	return nil, nil
}

func (v *DeploymentTargetValidator) validateDeploymentClassUniqueness(ctx context.Context, obj *DeploymentTarget) error {
	// This check does not provide a strong guarantee for unique deployment targets per deployment class in a landscape.
	// If two deployment targets with the same class are created in quick succession, the webhook may not be able to
	// detect the conflict. This is an accepted edge case for now because the restriction may be loosened in the future
	// anyway. At the moment, deployers can decide on their own how to handle the situation if multiple deployment
	// targets exist.

	list := &DeploymentTargetList{}
	if err := v.Client.List(ctx, list, client.InNamespace(obj.Namespace)); err != nil {
		return apierrors.NewInternalError(fmt.Errorf("failed to list DeploymentTargets in namespace %q: %w", obj.Namespace, err))
	}

	for i := range list.Items {
		existing := &list.Items[i]
		if existing.Name == obj.Name {
			continue
		}
		if existing.Spec.DeploymentClassName == obj.Spec.DeploymentClassName {
			return field.Invalid(
				field.NewPath("spec").Child("deploymentClassName"),
				obj.Spec.DeploymentClassName,
				fmt.Sprintf(
					"deployment class name must be unique across DeploymentTargets in namespace %q (already used by DeploymentTarget %q)",
					obj.Namespace,
					existing.Name,
				),
			)
		}
	}
	return nil
}
