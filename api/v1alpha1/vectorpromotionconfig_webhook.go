package v1alpha1

import (
	pkgwebhook "github.com/konfidence-project/konfidence/pkg/webhook"
	ctrl "sigs.k8s.io/controller-runtime"
)

// SetupVectorPromotionConfigWebhookWithManager registers the webhook with the manager.
func SetupVectorPromotionConfigWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &VectorPromotionConfig{}).
		WithValidator(pkgwebhook.NewProjectNamespaceValidator[*VectorPromotionConfig](mgr.GetClient(), VectorPromotionConfigKind)).
		Complete()
}

//nolint:lll // Kubebuilder annotations are intentionally long.
// +kubebuilder:webhook:path=/validate-konfidence-cloud-v1alpha1-vectorpromotionconfig,mutating=false,failurePolicy=fail,sideEffects=None,groups=konfidence.cloud,resources=vectorpromotionconfigs,verbs=create;update,versions=v1alpha1,name=vvectorpromotionconfig.konfidence.cloud,admissionReviewVersions=v1
