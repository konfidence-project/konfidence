package v1alpha1

// CredentialsConfig defines a credential reference to a secret or configMap used to access OCM backends (like OCI registry).
type CredentialsConfig struct {
	// Kind of the configuration resource. Allowed values are Secret or ConfigMap.
	Kind string `json:"kind"`

	// APIVersion is the api version of the of configuration resource, e.g. v1.
	APIVersion string `json:"apiVersion"`

	// Name is the name	 of the of configuration resource.
	Name string `json:"name"`
}
