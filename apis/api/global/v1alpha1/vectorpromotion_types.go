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
	// VectorPromotionKind is kind of the VectorPromotion resource.
	VectorPromotionKind = "VectorPromotion"

	// VectorPromotionStatePending indicates that the promotion has been created and is waiting to be executed.
	VectorPromotionStatePending = "Pending"
	// VectorPromotionStateRunning indicates that the promotion is currently being executed.
	VectorPromotionStateRunning = "Running"
	// VectorPromotionStateCompleted indicates that the promotion has been completed successfully.
	VectorPromotionStateCompleted = "Completed"
	// VectorPromotionStateFailed indicates that the promotion has failed.
	VectorPromotionStateFailed = "Failed"
)

// VectorPromotionSpec defines the desired state of VectorPromotion.
type VectorPromotionSpec struct {
	// VectorPromotionConfigRef is the name of the VectorPromotionConfig that defines the promotion flow to execute.
	// +kubebuilder:validation:MinLength=1
	VectorPromotionConfigRef string `json:"vectorPromotionConfigRef"`

	// TTL defines how long the VectorPromotion should be kept after completion.
	// Once the TTL expires after the promotion reaches a terminal state (Completed or Failed),
	// the resource is eligible for automatic deletion. If no TTL is set, no deletion happens.
	// +kubebuilder:validation:Optional
	TTL *metav1.Duration `json:"ttl,omitempty"`
}

// VectorPromotionStatus defines the observed state of VectorPromotion.
type VectorPromotionStatus struct {
	// State reflects the overall lifecycle state of the promotion.
	// +kubebuilder:validation:Enum=Pending;Running;Completed;Failed
	// +optional
	State      string             `json:"state,omitempty"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Config",type=string,JSONPath=".spec.vectorPromotionConfigRef",description="The referenced VectorPromotionConfig"
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=".status.state",description="The overall state of the promotion"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp",description="Age"

// VectorPromotion triggers a one-time execution of a promotion flow defined by a VectorPromotionConfig.
type VectorPromotion struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VectorPromotionSpec   `json:"spec,omitempty"`
	Status VectorPromotionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VectorPromotionList contains a list of VectorPromotion.
type VectorPromotionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VectorPromotion `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VectorPromotion{}, &VectorPromotionList{})
}
