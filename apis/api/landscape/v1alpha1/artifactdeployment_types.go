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
	ArtifactDeploymentKind = "ArtifactDeployment"

	// ArtifactDeploymentReadyCondition indicates that the artifact deployment is ready.
	ArtifactDeploymentReadyCondition = "ArtifactDeploymentReady"
)

// ArtifactDeploymentSpec defines the desired state of ArtifactDeployment.
type ArtifactDeploymentSpec struct {
	Type      string       `json:"type,omitempty"`
	Component OCMComponent `json:"component,omitempty"`
}

type OCMComponent struct {
	Name         string `json:"name"`
	Provider     string `json:"provider"`
	CreationTime string `json:"creationTime,omitempty"`
	Version      string `json:"version,omitempty"`

	Resources []OCMResource `json:"resources,omitempty"`
}

type OCMResource struct {
	Name    string `json:"name"`
	Image   string `json:"image"`
	Version string `json:"version"`
	Type    string `json:"type"`
}

// ArtifactDeploymentStatus defines the observed state of ArtifactDeployment.
type ArtifactDeploymentStatus struct {
	DeploymentResult *DeploymentResult  `json:"deploymentResult,omitempty"`
	Conditions       []metav1.Condition `json:"conditions,omitempty"`
}

type DeploymentResult struct {
	URL string `json:"url"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`

// ArtifactDeployment is the Schema for the artifactdeployments API.
type ArtifactDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ArtifactDeploymentSpec   `json:"spec,omitempty"`
	Status ArtifactDeploymentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ArtifactDeploymentList contains a list of ArtifactDeployment.
type ArtifactDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ArtifactDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ArtifactDeployment{}, &ArtifactDeploymentList{})
}
