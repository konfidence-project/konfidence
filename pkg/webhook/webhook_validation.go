package webhook

import (
	"context"
	"fmt"
	"strings"

	utils "github.com/konfidence-project/konfidence/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// validateNamespaceType is a common validation function that checks if a namespace exists
// and has the correct type and name labels.
func validateNamespaceType(ctx context.Context, c client.Client, namespace, expectedType, typeLabel, nameLabel string) error {
	ns := &corev1.Namespace{}
	if err := c.Get(ctx, types.NamespacedName{Name: namespace}, ns); err != nil {
		if apierrors.IsNotFound(err) {
			msg := fmt.Sprintf(
				"namespace does not exist. Create a %s resource and use its resulting namespace",
				strings.ToUpper(expectedType),
			)

			return field.Invalid(
				field.NewPath("metadata").Child("namespace"),
				namespace,
				msg,
			)
		}
		return apierrors.NewInternalError(fmt.Errorf("failed to get parent namespace %s: %w", namespace, err))
	}

	if ns.Labels == nil {
		msg := fmt.Sprintf(
			"namespace has no labels. Create a %s resource and use its resulting namespace (required label: %s=%s)",
			strings.ToUpper(expectedType),
			typeLabel,
			expectedType,
		)

		return field.Invalid(
			field.NewPath("metadata").Child("namespace"),
			namespace,
			msg,
		)
	}

	// Check type label
	nsType, hasType := ns.Labels[typeLabel]
	if !hasType || nsType != expectedType {
		msg := fmt.Sprintf(
			"namespace is not a %s namespace. Create a %s resource and use its resulting namespace (required label: %s=%s)",
			expectedType, strings.ToUpper(expectedType), typeLabel, expectedType,
		)

		return field.Invalid(
			field.NewPath("metadata").Child("namespace"),
			namespace,
			msg,
		)
	}

	// Check name label
	name, hasName := ns.Labels[nameLabel]
	if !hasName || name == "" {
		msg := fmt.Sprintf(
			"namespace is missing %s name. Create a %s resource and use its resulting namespace (required label: %s=<%s-name>)",
			expectedType, strings.ToUpper(expectedType), nameLabel, expectedType,
		)

		return field.Invalid(
			field.NewPath("metadata").Child("namespace"),
			namespace,
			msg,
		)
	}

	return nil
}

// ValidateProjectNamespace checks that a resource is created in a valid project namespace.
// This is shared validation logic for resources that must live in project namespaces.
func ValidateProjectNamespace(ctx context.Context, c client.Client, namespace string) error {
	return validateNamespaceType(ctx, c, namespace, "project", utils.ProjectTypeLabel, utils.ProjectNameLabel)
}

// ValidateLandscapeNamespace checks that a resource is created in a valid landscape namespace.
// This is shared validation logic for resources that must live in landscape namespaces.
func ValidateLandscapeNamespace(ctx context.Context, c client.Client, namespace string) error {
	return validateNamespaceType(ctx, c, namespace, "landscape", utils.ProjectTypeLabel, utils.LandscapeNameLabel)
}
