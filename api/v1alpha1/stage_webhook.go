package v1alpha1

import (
	pkgwebhook "github.com/konfidence-project/konfidence/pkg/webhook"
	ctrl "sigs.k8s.io/controller-runtime"
)

// SetupStageWebhookWithManager registers the webhook with the manager.
func SetupStageWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &Stage{}).
		WithValidator(pkgwebhook.NewLandscapeNamespaceValidator[*Stage](mgr.GetClient(), StageKind)).
		Complete()
}

//nolint:lll // Kubebuilder annotations are intentionally long.
// +kubebuilder:webhook:path=/validate-konfidence-cloud-v1alpha1-stage,mutating=false,failurePolicy=fail,sideEffects=None,groups=konfidence.cloud,resources=stages,verbs=create;update,versions=v1alpha1,name=vstage.konfidence.cloud,admissionReviewVersions=v1
