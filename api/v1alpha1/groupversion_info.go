// Package v1alpha1 contains API Schema definitions for the konfidence v1alpha1 API group.
// It covers both galaxy (management-plane) and star (workload-plane) kinds, which share
// a single API group; galaxy vs. star remains a code-organization convention (ADR-0022).
// +kubebuilder:object:generate=true
// +groupName=konfidence.cloud
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is group version used to register these objects.
	GroupVersion = schema.GroupVersion{Group: "konfidence.cloud", Version: "v1alpha1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	//nolint:staticcheck // scheme.Builder is deprecated but still the kubebuilder scaffold; all *_types.go register through it
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)
