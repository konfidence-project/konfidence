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
	// Type references a DeploymentClass by its spec.type field.
	// The referenced DeploymentClass must exist in the cluster.
	// This determines which controller will handle deployments to this target.
	// The value must be unique across DeploymentTargets in the same namespace.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*\/[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	Type string `json:"type"`

	// Connection defines how to connect to this deployment target.
	// The structure and interpretation of connection details is specific to the
	// deployment class and its implementing controller.
	// +kubebuilder:validation:Required
	Connection DeploymentTargetConnection `json:"connection"`
}

// DeploymentTargetConnection defines connection information for a deployment target.
type DeploymentTargetConnection struct {
	// Type is a hint for how to interpret the connection reference. It is advisory
	// and informational only. The deployer controller interprets and enforces its meaning.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=64
	Type string `json:"type"`

	// Ref references a Secret or ConfigMap containing connection details.
	// The structure of the referenced resource depends on the connection type
	// and the deployment class requirements.
	// +optional
	Ref *ConnectionRef `json:"ref,omitempty"`
}

// ConnectionRef identifies a Secret or ConfigMap in the same namespace.
type ConnectionRef struct {
	// APIGroup is the group for the resource being referenced.
	// Defaults to the core API group ("") for Secret and ConfigMap.
	// For deployer-specific CRDs, set this to the appropriate API group.
	// +optional
	APIGroup string `json:"apiGroup,omitempty"`

	// Kind is the resource kind (e.g., "Secret", "ConfigMap", or a deployer-specific kind).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=64
	Kind string `json:"kind"`

	// Name is the name of the referenced resource.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Name string `json:"name"`
}

// DeploymentTargetStatus defines the observed state of DeploymentTarget.
// The deployer controller responsible for this target's DeploymentClass is expected
// to set the Ready condition once it has accepted the resource. What "accepted" means
// is up to the deployer. It may include connectivity checks or simply validate the
// configuration.
type DeploymentTargetStatus struct {
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//nolint:lll // Kubebuilder annotations are intentionally long.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories=konfidence;kden
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`,description="Deployment class type"
// +kubebuilder:printcolumn:name="Connection",type=string,JSONPath=`.spec.connection.type`,description="Connection type"
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`,description="Ready status"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// DeploymentTarget is the Schema for the deploymenttargets API. A DeploymentTarget
// configures a concrete deployment destination within a landscape for a specific
// deployment class. It is namespace-scoped and created in landscape namespaces.
// Multiple DeploymentTargets can exist in the same landscape, but their types must
// be unique within the namespace.
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
