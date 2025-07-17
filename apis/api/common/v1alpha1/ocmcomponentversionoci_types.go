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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// OCMComponentVersionOCIKind is the kind of the OCMComponentVersionOCI resource.
	OCMComponentVersionOCIKind = "OCMComponentVersionOCI"
)

// OCMComponentVersionOCISpec defines the desired state of OCMComponentVersionOCI.
type OCMComponentVersionOCISpec struct {
	// Component specifies the name of the ComponentVersion.
	// +required
	Component string `json:"component"`

	// Version specifies the version information for the ComponentVersion.
	// +required
	Version string `json:"version"`

	// Repository provides details about the OCI repository from which the component
	// descriptor can be retrieved.
	// +required
	Repository Repository `json:"repository"`
}

// Repository specifies access details for the repository that contains OCM ComponentVersions.
type Repository struct {
	// URL specifies the URL of the OCI registry in which the ComponentVersion is stored.
	// +required
	URL string `json:"url"`

	// SecretRef specifies the credentials used to access the OCI registry.
	// +optional
	SecretRef *corev1.LocalObjectReference `json:"secretRef,omitempty"`
}

// OCMComponentVersionOCIStatus defines the observed state of OCMComponentVersionOCI.
type OCMComponentVersionOCIStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// OCMComponentVersionOCI is the Schema for the ocmcomponentversionocis API.
type OCMComponentVersionOCI struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OCMComponentVersionOCISpec   `json:"spec,omitempty"`
	Status OCMComponentVersionOCIStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OCMComponentVersionOCIList contains a list of OCMComponentVersionOCI.
type OCMComponentVersionOCIList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OCMComponentVersionOCI `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OCMComponentVersionOCI{}, &OCMComponentVersionOCIList{})
}
