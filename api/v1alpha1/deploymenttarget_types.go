package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// DeploymentTargetKind is kind of the DeploymentTarget resource.
	DeploymentTargetKind = "DeploymentTarget"

	// DeploymentTargetReadyCondition is the Ready condition for the DeploymentTarget resource.
	// It is set by the deployer controller once it has accepted the resource.
	DeploymentTargetReadyCondition = "Ready"
)

// DeploymentTargetSpec defines the desired state of DeploymentTarget.
type DeploymentTargetSpec struct {
	// ArtifactType identifies which type of artifact should be deployed to the specified target. The ArtifactType is
	// selected by a deployer whose controllers are responsible for reconciling the DeploymentTarget resource. If there
	// is no deployer covering the given ArtifactType, the DeploymentTarget will never become ready. ArtifactType is
	// immutable because changing it may transfer ownership of the DeploymentTarget to a different controller.
	//
	// Currently, Konfidence only supports one DeploymentTarget per ArtifactType in a landscape, so this field should
	// be unique across all DeploymentTargets in the same landscape namespace. If multiple DeploymentTargets with the
	// same ArtifactType exist in a single landscape namespace, deployers may fail their deployments, as they cannot
	// determine which DeploymentTarget takes precedence.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="artifactType is immutable"
	ArtifactType string `json:"artifactType"`

	// Parameters define how to connect to the target runtime described by this DeploymentTarget. It may point to
	// deployer-specific configuration containing connection details, credentials and further settings required to
	// connect to the target runtime. Their structure and interpretation are defined by the deployer that owns ArtifactType.
	// +optional
	Parameters *Parameters `json:"parameters,omitempty"`
}

// Parameters reference a resource containing deployer-specific configuration for connecting to the target runtime.
// The referenced resource's structure and interpretation are defined by the deployer responsible for the
// DeploymentTarget's ArtifactType.
type Parameters struct {
	// APIGroup is the group for the resource being referenced.
	// Defaults to the core API group ("") for Secret and ConfigMap.
	// For deployer-specific CRDs, set this to the appropriate API group.
	// +optional
	APIGroup string `json:"apiGroup,omitempty"`

	// Kind is the resource kind (e.g., "Secret", "ConfigMap", or a deployer-specific kind).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	Kind string `json:"kind"`

	// Name is the name of the referenced resource.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Name string `json:"name"`
}

// DeploymentTargetStatus defines the state observed by the responsible deployer.
type DeploymentTargetStatus struct {
	// Controller identifies the deployer and release version reporting this status.
	// +optional
	Controller string `json:"controller,omitempty"`

	// +optional
	// +listType=map
	// +listMapKey=type
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//nolint:lll // Kubebuilder annotations are intentionally long.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories=konfidence;kden
// +kubebuilder:printcolumn:name="Artifact Type",type=string,JSONPath=`.spec.artifactType`,description="Artifact type"
// +kubebuilder:printcolumn:name="Controller",type=string,JSONPath=`.status.controller`,description="Reporting deployer"
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`,description="Ready status"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// DeploymentTarget is the Schema for the deploymenttargets API. A DeploymentTarget
// configures a concrete deployment destination within a landscape for a specific
// artifact type. It is namespace-scoped and created in landscape namespaces.
// At most one DeploymentTarget for a given artifact type is allowed in a namespace.
type DeploymentTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeploymentTargetSpec   `json:"spec,omitempty"`
	Status DeploymentTargetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DeploymentTargetList contains a list of DeploymentTarget.
type DeploymentTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DeploymentTarget `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DeploymentTarget{}, &DeploymentTargetList{})
}
