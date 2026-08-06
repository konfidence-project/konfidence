package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// DeploymentClassKind is kind of the DeploymentClass resource.
	DeploymentClassKind = "DeploymentClass"
)

// DeploymentClassSpec defines the desired state of DeploymentClass.
type DeploymentClassSpec struct {
	// Type is the unique deployment class type identifier (e.g., "konfidence.cloud/helm").
	// This is what artifacts reference in their manifest.type field to select which
	// deployer handles their deployment. The type must be unique across all DeploymentClasses
	// in the cluster and follows the pattern "<vendor-domain>/<class-name>".
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*\/[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	Type string `json:"type"`

	// Controller is the name of the controller that implements this deployment class.
	// This identifies which operator/controller is responsible for reconciling
	// resources of this deployment class (e.g., "kubernetes-landscape-orchestrator").
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Controller string `json:"controller"`
}

//nolint:lll // Kubebuilder annotations are intentionally long.
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,categories=konfidence;kden
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`,description="Deployment class type"
// +kubebuilder:printcolumn:name="Controller",type=string,JSONPath=`.spec.controller`,description="Controller name"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// DeploymentClass is the Schema for the deploymentclasses API. A DeploymentClass
// declares a deployment capability provided by a deployer (controller). It is a
// cluster-scoped resource installed by deployers to advertise their capabilities.
// Artifacts reference the spec.type field to select which deployer handles their deployment.
// The type must be unique across all DeploymentClasses in the cluster.
type DeploymentClass struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec DeploymentClassSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// DeploymentClassList contains a list of DeploymentClass.
type DeploymentClassList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DeploymentClass `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DeploymentClass{}, &DeploymentClassList{})
}
