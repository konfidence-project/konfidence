package v1alpha1

import "k8s.io/apimachinery/pkg/runtime"

// VectorConfig defines feature flags and authored configuration values for a vector.
type VectorConfig struct {
	// Features define the feature flags.
	Features *runtime.RawExtension `json:"features,omitempty"`

	// Authored define the authored configuration values.
	Authored *runtime.RawExtension `json:"authored,omitempty"`
}
