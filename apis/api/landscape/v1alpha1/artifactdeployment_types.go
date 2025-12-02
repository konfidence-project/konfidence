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
	// ArtifactDeploymentKind is the kind for ArtifactDeployment resources.
	ArtifactDeploymentKind = "ArtifactDeployment"

	// ArtifactFetchedCondition indicates that the deployer was able to successfully download the artifact.
	ArtifactFetchedCondition = "ArtifactFetched"

	// ArtifactDeployedCondition indicates that the deployer was able to successfully deploy the artifact.
	ArtifactDeployedCondition = "ArtifactDeployed"

	// AppHealthyCondition indicates that the health check of the application was successful.
	AppHealthyCondition = "AppHealthy"

	// DeploymentResultCreatedCondition indicates that the fetching the DeploymentResult from the cluster was successful.
	DeploymentResultCreatedCondition = "DeploymentResultCreated"

	// ArtifactDeploymentReadyCondition indicates that the resource was successfully reconciled.
	ArtifactDeploymentReadyCondition = "Ready"
)

// ArtifactDeploymentSpec defines the desired state of an ArtifactDeployment. It describes the artifact to be deployed,
// optional post-deployment tasks, and optional metadata derived from an OCM ComponentVersion. A deployer interprets
// the specification according to the artifact type in Manifest.Type.
type ArtifactDeploymentSpec struct {
	// Manifest contains information about the artifact itself and the deployer implementation responsible for handling it.
	Manifest ArtifactManifest `json:"manifest"`

	// TaskManifests describes optional post-deployment tasks (commonly used for vector migrations such as database
	// schema updates). Tasks are executed after the artifact has been deployed and may form a dependency graph via
	// DependsOn.
	TaskManifests []TaskManifest `json:"taskManifests"`

	// Component contains OCM metadata associated with the artifact. This is a simplified mapping of the OCM ComponentVersion.
	Component OCMComponent `json:"component,omitempty"`
}

// OCMComponent is a wrapper around the OCM ComponentVersion. It can be used to attach additional metadata to an
// ArtifactDeployment. The component may include one or more OCM resources.
type OCMComponent struct {
	// Name is the OCM ComponentVersion name.
	Name string `json:"name"`

	// Version is the OCM ComponentVersion version.
	// +optional
	Version string `json:"version,omitempty"`

	// Resources contains OCM resources belonging to this component. The structure is intentionally generic to support
	// the requirements of deployers targeting different runtimes.
	// +optional
	Resources []OCMResource `json:"resources,omitempty"`
}

// OCMResource represents a single resource of an OCM ComponentVersion. The content and type are deployer-specific and
// opaque to the API.
type OCMResource struct {
	// Name is the resource name.
	Name string `json:"name"`

	// Content holds raw resource data, typically an embedded manifest, file, or
	// binary payload.
	Content runtime.RawExtension `json:"content"`

	// Type describes the resource type, following OCM conventions.
	Type string `json:"type"`
}

// ArtifactDeploymentStatus defines the observed state of ArtifactDeployment.
type ArtifactDeploymentStatus struct {
	// ObservedGeneration is the last observed generation.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions describes the state of the deployment lifecycle. The following conditions are expected:
	//
	//   - ArtifactFetched: the artifact was successfully retrieved
	//   - ArtifactDeployed: the artifact was successfully deployed
	//   - AppHealthy: the deployer reports the workload as healthy
	//
	// Conditions progress in a linear order:
	// ArtifactFetched -> ArtifactDeployed -> AppHealthy
	//
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// DeploymentResults captures structured outputs produced by the deployer during the deployment process—such as
	// computed DNS names, service endpoints, generated configuration, or other workload-specific details.
	//
	// Results should be treated as immutable for a given generation and may be consumed by later stages of a vector
	// rollout (e.g., routing configuration).
	//
	// Each result must have a unique Name.
	// +optional
	DeploymentResults []DeploymentResult `json:"deploymentResult,omitempty"`
}

// DeploymentResult contains a single output produced by a deployer. These results are used to transport information
// from the deployer to later phases of the vector lifecycle.
type DeploymentResult struct {
	// Name is a unique identifier for the result within an ArtifactDeploymentStatus.
	Name string `json:"name"`

	// Type describes the structure contained in Spec. Each deployer may define multiple result types.
	Type string `json:"type"`

	// Spec contains deployer-specific structured data. Its format is determined by the Type field.
	Spec runtime.RawExtension `json:"spec"`
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

// ArtifactManifest describes the content of the artifact, thus it determines the deployer implementation responsible
// for handling it.
type ArtifactManifest struct {
	// Type specifies the deployer that should handle this artifact (e.g., "cloud.konfidence.flux.helm",
	// "cloud.konfidence.flux.kustomize", or any custom deployer type). Deployers implement their own interpretation
	// of the artifact's contents.
	Type string `json:"type"`

	// AllowReuse indicates whether the deployed artifact instance may be shared across multiple VectorDeployments.
	// Reuse allows more efficient resource consumption but requires the artifact to be independent of vector-specific
	// runtime context.
	AllowReuse bool `json:"allowReuse"`
}

// TaskManifest defines a post-deployment task that is executed after the artifact has been deployed. Tasks are
// commonly used for vector migrations (such as database schema changes) but may represent any post-deployment action.
//
// Tasks form a directed acyclic graph (DAG) at the *vector level* rather than only within a single ArtifactDeployment.
// A task may depend on tasks belonging to other microservices or artifacts in the same VectorDeployment. These
// cross-artifact dependencies allow defining a globally ordered migration or transformation workflow.
//
// The controller responsible for the task type interprets the Spec field and performs the execution once all declared
// dependencies have completed successfully.
type TaskManifest struct {
	// Name uniquely identifies this task within the entire vector. This name may be referenced by other tasks across
	// different artifacts.
	Name string `json:"name"`

	// Type specifies the task controller or execution runtime (e.g. "k8s-job", or any custom task runtime). Different
	// task types correspond to different task controllers, each interpreting the Spec field according to their own semantics.
	Type string `json:"type"`

	// DependsOn lists names of other tasks that must complete before this task may run. Dependencies may reference
	// tasks within the same artifact or any other artifact that participates in the same VectorDeployment, allowing the
	// formation of a vector-wide DAG.
	// +optional
	DependsOn []string `json:"dependsOn,omitempty"`

	// Spec contains task-specific configuration. The structure depends on the task Type and is interpreted by the
	// corresponding task controller.
	Spec runtime.RawExtension `json:"spec"`
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
