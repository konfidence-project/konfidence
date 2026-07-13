package v1alpha1

// Credentials holds credentials for various purposes — for example OCM
// repository access and signing/verification key material.
//
// +kubebuilder:validation:XValidation:rule="has(self.ocm)",message="at least one credential surface must be set"
type Credentials struct {
	// +optional
	OCM *OCMCredentials `json:"ocm,omitempty"`
}

// OCMCredentials lists Secrets holding `.ocmconfig` or `.dockerconfigjson` data.
// All references are same-namespace.
type OCMCredentials struct {
	// +kubebuilder:validation:MinItems=1
	Refs []CredentialRef `json:"refs"`
}

// CredentialRef references a Secret in the same namespace as the holding resource.
type CredentialRef struct {
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}
