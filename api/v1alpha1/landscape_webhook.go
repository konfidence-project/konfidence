package v1alpha1

import (
	pkgwebhook "github.com/konfidence-project/konfidence/pkg/webhook"
	ctrl "sigs.k8s.io/controller-runtime"
)

// SetupLandscapeWebhookWithManager registers the webhook with the manager.
func SetupLandscapeWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &Landscape{}).
		WithValidator(pkgwebhook.NewProjectNamespaceValidator[*Landscape](mgr.GetClient(), LandscapeKind)).
		Complete()
}

//nolint:lll // Kubebuilder annotations are intentionally long.
// +kubebuilder:webhook:path=/validate-konfidence-cloud-v1alpha1-landscape,mutating=false,failurePolicy=fail,sideEffects=None,groups=konfidence.cloud,resources=landscapes,verbs=create;update,versions=v1alpha1,name=vlandscape.konfidence.cloud,admissionReviewVersions=v1
