package v1alpha1

import (
	pkgwebhook "github.com/konfidence-project/konfidence/pkg/webhook"
	ctrl "sigs.k8s.io/controller-runtime"
)

// SetupStageConfigurationWebhookWithManager registers the webhook with the manager.
func SetupStageConfigurationWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &StageConfiguration{}).
		WithValidator(pkgwebhook.NewProjectNamespaceValidator[*StageConfiguration](mgr.GetClient(), StageConfigurationKind)).
		Complete()
}

//nolint:lll // Kubebuilder annotations are intentionally long.
// +kubebuilder:webhook:path=/validate-konfidence-cloud-v1alpha1-stageconfiguration,mutating=false,failurePolicy=fail,sideEffects=None,groups=konfidence.cloud,resources=stageconfigurations,verbs=create;update,versions=v1alpha1,name=vstageconfiguration.konfidence.cloud,admissionReviewVersions=v1
