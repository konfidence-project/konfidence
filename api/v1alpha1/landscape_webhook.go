package v1alpha1

import (
	"context"
	"fmt"

	pkgctrl "github.com/konfidence-project/konfidence/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var landscapelog = logf.Log.WithName("landscape-webhook")

// SetupLandscapeWebhookWithManager registers the webhook with the manager.
func SetupLandscapeWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &Landscape{}).
		WithValidator(&LandscapeValidator{Client: mgr.GetClient()}).
		Complete()
}

//nolint:lll // Kubebuilder annotations are intentionally long.
// +kubebuilder:webhook:path=/validate-konfidence-cloud-v1alpha1-landscape,mutating=false,failurePolicy=fail,sideEffects=None,groups=konfidence.cloud,resources=landscapes,verbs=create;update,versions=v1alpha1,name=vlandscape.konfidence.cloud,admissionReviewVersions=v1

// LandscapeValidator validates Landscape resources.
// +kubebuilder:object:generate=false
type LandscapeValidator struct {
	Client client.Client
}

// ValidateCreate validates a Landscape on creation.
func (v *LandscapeValidator) ValidateCreate(ctx context.Context, landscape *Landscape) (admission.Warnings, error) {
	landscapelog.Info("validating Landscape creation", "name", landscape.Name, "namespace", landscape.Namespace)

	if err := v.validateParentNamespace(ctx, landscape); err != nil {
		return nil, err
	}

	return nil, nil
}

// ValidateUpdate validates a Landscape on update.
func (v *LandscapeValidator) ValidateUpdate(ctx context.Context, oldLandscape, landscape *Landscape) (admission.Warnings, error) {
	landscapelog.Info("validating Landscape update", "name", landscape.Name, "namespace", landscape.Namespace)

	if err := v.validateParentNamespace(ctx, landscape); err != nil {
		return nil, err
	}

	return nil, nil
}

// ValidateDelete validates a Landscape on deletion (always allow).
func (v *LandscapeValidator) ValidateDelete(ctx context.Context, landscape *Landscape) (admission.Warnings, error) {
	return nil, nil
}

// validateParentNamespace checks that the Landscape is created in a valid project namespace.
func (v *LandscapeValidator) validateParentNamespace(ctx context.Context, landscape *Landscape) error {
	parentNS := &corev1.Namespace{}
	if err := v.Client.Get(ctx, types.NamespacedName{Name: landscape.Namespace}, parentNS); err != nil {
		if apierrors.IsNotFound(err) {
			return field.Invalid(
				field.NewPath("metadata").Child("namespace"),
				landscape.Namespace,
				"namespace does not exist. Create a Project resource and use its resulting namespace",
			)
		}
		return apierrors.NewInternalError(fmt.Errorf("failed to get parent namespace %s: %w", landscape.Namespace, err))
	}

	if parentNS.Labels == nil {
		return field.Invalid(
			field.NewPath("metadata").Child("namespace"),
			landscape.Namespace,
			fmt.Sprintf("namespace has no labels. Create a Project resource and use its resulting namespace (required label: %s=project)", pkgctrl.ProjectTypeLabel),
		)
	}

	nsType, hasType := parentNS.Labels[pkgctrl.ProjectTypeLabel]
	if !hasType || nsType != "project" {
		msg := fmt.Sprintf(
			"namespace is not a project namespace. Create a Project resource and use its resulting namespace (required label: %s=project)",
			pkgctrl.ProjectTypeLabel,
		)

		return field.Invalid(
			field.NewPath("metadata").Child("namespace"),
			landscape.Namespace,
			msg,
		)
	}

	projectName, hasProject := parentNS.Labels[pkgctrl.ProjectNameLabel]
	if !hasProject || projectName == "" {
		msg := fmt.Sprintf(
			"namespace is missing project name. Create a Project resource and use its resulting namespace (required label: %s=<project-name>)",
			pkgctrl.ProjectNameLabel,
		)

		return field.Invalid(
			field.NewPath("metadata").Child("namespace"),
			landscape.Namespace,
			msg,
		)
	}

	return nil
}
