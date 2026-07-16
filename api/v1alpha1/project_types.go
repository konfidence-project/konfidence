package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ProjectKind is kind of the Project resource.
	ProjectKind = "Project"

	// ProjectReadyCondition is the ready condition for the Project resource.
	ProjectReadyCondition = "Ready"
	// ProjectNamespaceReadyCondition reports on the managed project namespace.
	ProjectNamespaceReadyCondition = "NamespaceReady"

	// ProjectReconciledReason indicates that the project was fully reconciled.
	ProjectReconciledReason = "ProjectReconciled"
	// ProjectTerminatingReason indicates that the project is being deleted and is
	// waiting for its namespace to terminate.
	ProjectTerminatingReason = "Terminating"
	// ProjectNamespaceReconciledReason indicates that the project namespace is in the desired state.
	ProjectNamespaceReconciledReason = "NamespaceReconciled"
	// ProjectNamespaceConflictReason indicates that the project namespace exists but is not managed by the project.
	ProjectNamespaceConflictReason = "NamespaceConflict"
	// ProjectNamespaceTerminatingReason indicates that the project namespace is currently terminating.
	ProjectNamespaceTerminatingReason = "NamespaceTerminating"
	// ProjectNamespaceCreateFailedReason indicates that creating or updating the project namespace failed.
	ProjectNamespaceCreateFailedReason = "NamespaceCreateFailed"

	// ProjectNamespacePrefix is the prefix of the default project namespace name.
	ProjectNamespacePrefix = "kden-project-"
)

// ProjectSpec defines the desired state of Project.
//
// The transition rule catches namespace being set or unset after creation;
// changing a set namespace is caught by the field-level rule. The two rules
// are split to stay within the CEL cost budget of the schema.
// +kubebuilder:validation:XValidation:rule="has(self.namespace) == has(oldSelf.namespace)",message="namespace is immutable"
type ProjectSpec struct {
	// DisplayName is the human-readable name of the project, shown in user
	// interfaces. It does not affect the namespace name or any label, and it
	// may be changed at any time.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Namespace overrides the name of the namespace created for this project.
	// When unset it defaults to "kden-project-<project-name>". It is immutable
	// once the Project exists, because the namespace and everything it holds
	// are bound to this name.
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="namespace is immutable"
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// RoleBindings grants project roles to callers. It maps a role name to the
	// list of subjects that hold that role; a caller holds the role if any
	// subject in the list matches (OR). The role names are a fixed, well-known
	// set for now (for example "admin", "pm", "dev"), but the field is a map so
	// the set can be extended without a schema change. See the Project
	// multi-tenancy ADR for the meaning of each role and the authorization flow.
	// RoleBindings is currently schema-only: no authorization is enforced yet.
	// +kubebuilder:validation:MaxProperties=32
	// +optional
	RoleBindings map[string]Subjects `json:"roleBindings,omitempty"`
}

// Subjects is the list of subjects that hold a role. A caller holds the role
// if any subject matches (OR).
// +kubebuilder:validation:MinItems=1
// +kubebuilder:validation:MaxItems=32
type Subjects []Subject

// Subject identifies who is granted a role. Exactly one identity source
// (session or jwks) must be set.
// +kubebuilder:validation:XValidation:rule="has(self.session) != has(self.jwks)",message="exactly one of session or jwks must be set"
type Subject struct {
	// Session matches an interactively authenticated user by group membership,
	// for example a person signed in through the identity provider.
	// +optional
	Session *SessionSubject `json:"session,omitempty"`

	// JWKS matches a workload identity presenting a token signed by a trusted
	// OIDC provider, for example a CI pipeline's OIDC token.
	// +optional
	JWKS *JWKSSubject `json:"jwks,omitempty"`
}

// SessionSubject matches an interactive user by group membership.
type SessionSubject struct {
	// MemberOf lists the groups that grant the role. Membership in any one of
	// the listed groups is sufficient to match (OR).
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=64
	// +kubebuilder:validation:items:MaxLength=253
	MemberOf []string `json:"memberOf"`
}

// JWKSSubject matches a workload token issued by a trusted OIDC provider,
// narrowed to a required audience and at least one token claim.
type JWKSSubject struct {
	// Endpoint is the OIDC discovery endpoint (the provider's
	// ".well-known/openid-configuration" URL) used to resolve the signing keys
	// that the presented token is verified against.
	// +kubebuilder:validation:MaxLength=2048
	// +kubebuilder:validation:Pattern=`^https://.*$`
	Endpoint string `json:"endpoint"`

	// Audience is the value the token's "aud" claim must carry. It is required
	// so that a token minted for a different service cannot be replayed against
	// Konfidence: the token is accepted only if it was issued for this audience.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=512
	Audience string `json:"audience"`

	// Claims narrows the match to tokens whose claims match the given patterns.
	// It maps a claim name (for example "sub", "repository" or "ref") to a
	// glob pattern the claim value must match; all listed claims must match
	// (AND). At least one claim is required so a subject cannot inadvertently
	// match every token a provider issues.
	// +kubebuilder:validation:MinProperties=1
	// +kubebuilder:validation:MaxProperties=32
	Claims map[string]GlobMatch `json:"claims"`
}

// GlobMatch is a claim-value match pattern using glob semantics, where "*"
// matches any run of characters (for example "repo:konfidence-project/*").
// +kubebuilder:validation:MaxLength=512
type GlobMatch string

// ProjectStatus defines the observed state of Project.
type ProjectStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Namespace is the name of the namespace managed for this project.
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

//nolint:lll // Kubebuilder annotations are intentionally long.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories=konfidence;kden
// +kubebuilder:validation:XValidation:rule="size(self.metadata.name) <= 50",message="project name must be at most 50 characters"
// +kubebuilder:printcolumn:name="Display Name",type=string,JSONPath=`.spec.displayName`
// +kubebuilder:printcolumn:name="Namespace",type=string,JSONPath=`.status.namespace`,description="The project namespace"
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Project is the Schema for the projects API. A Project owns a dedicated
// namespace that stores the project's Galaxy resources. The project name is
// capped at 50 characters so the derived namespace name and label values
// stay within the 63-character Kubernetes limits.
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectSpec   `json:"spec,omitempty"`
	Status ProjectStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProjectList contains a list of Project.
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Project{}, &ProjectList{})
}
