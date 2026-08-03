// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Konfidence contributors
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// NewProjectNamespaceValidator creates a new namespaceValidator that validates if resources
// are created inside a project namespace. Project namespaces are identified by having
// the project type label and project name label set.
func NewProjectNamespaceValidator[T client.Object](c client.Client, kind string) *namespaceValidator[T] {
	return &namespaceValidator[T]{
		Client:       c,
		ValidateFunc: ValidateProjectNamespace,
		ResourceKind: kind,
	}
}

// NewLandscapeNamespaceValidator creates a new namespaceValidator that validates if resources
// are created inside a landscape namespace. Landscape namespaces are identified by having
// the landscape type label and landscape name label set.
func NewLandscapeNamespaceValidator[T client.Object](c client.Client, kind string) *namespaceValidator[T] {
	return &namespaceValidator[T]{
		Client:       c,
		ValidateFunc: ValidateLandscapeNamespace,
		ResourceKind: kind,
	}
}

type validateNamespaceFunc func(ctx context.Context, c client.Client, namespace string) error

type namespaceValidator[T client.Object] struct {
	Client       client.Client
	ValidateFunc validateNamespaceFunc
	ResourceKind string
}

func (v *namespaceValidator[T]) ValidateCreate(ctx context.Context, obj T) (admission.Warnings, error) {
	log := logf.FromContext(ctx).WithName(v.ResourceKind + "-webhook")
	log.Info("validating creation", "name", obj.GetName(), "namespace", obj.GetNamespace())

	if v.ValidateFunc != nil {
		if err := v.ValidateFunc(ctx, v.Client, obj.GetNamespace()); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (v *namespaceValidator[T]) ValidateUpdate(ctx context.Context, _, obj T) (admission.Warnings, error) {
	log := logf.FromContext(ctx).WithName(v.ResourceKind + "-webhook")
	log.Info("validating update", "name", obj.GetName(), "namespace", obj.GetNamespace())

	if v.ValidateFunc != nil {
		if err := v.ValidateFunc(ctx, v.Client, obj.GetNamespace()); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (v *namespaceValidator[T]) ValidateDelete(_ context.Context, _ T) (admission.Warnings, error) {
	return nil, nil
}
