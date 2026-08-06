package v1alpha1

import (
	pkgwebhook "github.com/konfidence-project/konfidence/pkg/webhook"
	ctrl "sigs.k8s.io/controller-runtime"
)

// SetupDeploymentTargetWebhookWithManager registers the webhook with the manager.
func SetupDeploymentTargetWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &DeploymentTarget{}).
		WithValidator(pkgwebhook.NewLandscapeNamespaceValidator[*DeploymentTarget](mgr.GetClient(), DeploymentTargetKind)).
		Complete()
}

//nolint:lll // Kubebuilder annotations are intentionally long.
// +kubebuilder:webhook:path=/validate-konfidence-cloud-v1alpha1-deploymenttarget,mutating=false,failurePolicy=fail,sideEffects=None,groups=konfidence.cloud,resources=deploymenttargets,verbs=create;update,versions=v1alpha1,name=vdeploymenttarget.konfidence.cloud,admissionReviewVersions=v1
