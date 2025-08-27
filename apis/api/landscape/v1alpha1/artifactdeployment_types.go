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
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	ArtifactDeploymentKind = "ArtifactDeployment"

	// ArtifactFetchedCondition indicates that the deployer was able to successfully download the artifact.
	ArtifactFetchedCondition = "ArtifactFetched"
	// ArtifactDeployedCondition indicates that the deployer was able to successfully deploy the artifact.
	ArtifactDeployedCondition = "ArtifactDeployed"
	// AppHealthyCondition indicates that the health check of the application was successful.
	AppHealthyCondition = "AppHealthy"
)

// ArtifactDeploymentSpec defines the desired state of ArtifactDeployment.
type ArtifactDeploymentSpec struct {
	Manifest       ArtifactManifest `json:"manifest"`
	TaskManifests  []TaskManifest   `json:"taskManifests"`
	ArtifactOcmRef string           `json:"artifactOcmRef"`
	ArtifactOcm    string           `json:"artifactOcm"`
}

// ArtifactDeploymentStatus defines the observed state of ArtifactDeployment.
type ArtifactDeploymentStatus struct {
	// ObservedGeneration is the last observed generation.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions holds the conditions for the ArtifactDeployment.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	DeploymentResult *DeploymentResult `json:"deploymentResult,omitempty"`
}

type DeploymentResult struct {
	URL string `json:"url"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Fetched",type="string",JSONPath=".status.conditions[?(@.type==\"ArtifactFetched\")].status",description=""
// +kubebuilder:printcolumn:name="Deployed",type="string",JSONPath=".status.conditions[?(@.type==\"ArtifactDeployed\")].status",description=""
// +kubebuilder:printcolumn:name="Healthy",type="string",JSONPath=".status.conditions[?(@.type==\"AppHealthy\")].message",description=""

// ArtifactDeployment is the Schema for the artifactdeployments API.
type ArtifactDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ArtifactDeploymentSpec   `json:"spec,omitempty"`
	Status ArtifactDeploymentStatus `json:"status,omitempty"`
}

type ArtifactManifest struct {
	Type       string `json:"type"`
	AllowReuse bool   `json:"allowReuse"`
}

type TaskManifest struct {
	Name      string               `json:"name"`
	Type      string               `json:"type"`
	DependsOn []string             `json:"dependsOn,omitempty"`
	Spec      runtime.RawExtension `json:"spec"`
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
