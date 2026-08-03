package v1alpha1

import (
	pkgwebhook "github.com/konfidence-project/konfidence/pkg/webhook"
	ctrl "sigs.k8s.io/controller-runtime"
)

// SetupVectorTemplateWebhookWithManager registers the webhook with the manager.
func SetupVectorTemplateWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &VectorTemplate{}).
		WithValidator(pkgwebhook.NewProjectNamespaceValidator[*VectorTemplate](mgr.GetClient(), VectorTemplateKind)).
		Complete()
}

//nolint:lll // Kubebuilder annotations are intentionally long.
// +kubebuilder:webhook:path=/validate-konfidence-cloud-v1alpha1-vectortemplate,mutating=false,failurePolicy=fail,sideEffects=None,groups=konfidence.cloud,resources=vectortemplates,verbs=create;update,versions=v1alpha1,name=vvectortemplate.konfidence.cloud,admissionReviewVersions=v1
