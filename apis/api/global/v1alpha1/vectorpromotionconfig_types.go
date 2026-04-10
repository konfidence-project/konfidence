/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// VectorPromotionConfigKind is kind of the VectorPromotionConfig resource.
	VectorPromotionConfigKind = "VectorPromotionConfig"
)

// VectorPromotionConfigSpec defines the desired state of VectorPromotionConfig.
type VectorPromotionConfigSpec struct {
	// Source is the OCM component reference to promote from.
	// This usually points to a version alias (e.g. :latest) that resolves to the component version to be promoted.
	// +kubebuilder:validation:MinLength=1
	Source string `json:"source"`

	// Target is the OCM component reference to promote to.
	// This usually points to a version alias (e.g. :promoted). The actual version string is taken from the source component version.
	// +kubebuilder:validation:MinLength=1
	Target string `json:"target"`
}

// VectorPromotionConfigStatus defines the observed state of VectorPromotionConfig.
type VectorPromotionConfigStatus struct {
	// LastPromotionTime reflects the last successful execution of a promotion with this configuration.
	// +optional
	LastPromotionTime *metav1.Time       `json:"lastPromotionTime,omitempty"`
	Conditions        []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.spec) || has(self.spec)", message="Spec is required once set"
// +kubebuilder:printcolumn:name="Source",type=string,JSONPath=".spec.source",description="The source OCM component reference"
// +kubebuilder:printcolumn:name="Target",type=string,JSONPath=".spec.target",description="The target OCM component reference"

// VectorPromotionConfig describes a promotion flow for a vector between a source and a target.
type VectorPromotionConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="VectorPromotionConfig spec is immutable after it has been set"
	// Spec defines the desired state of the VectorPromotionConfig and is immutable after it has been set
	Spec   VectorPromotionConfigSpec   `json:"spec,omitempty"`
	Status VectorPromotionConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VectorPromotionConfigList contains a list of VectorPromotionConfig.
type VectorPromotionConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VectorPromotionConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VectorPromotionConfig{}, &VectorPromotionConfigList{})
}
