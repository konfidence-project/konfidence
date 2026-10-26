package v1alpha1

// NamespacePolicy controls how a controller takes ownership of its managed namespace.
type NamespacePolicy string

const (
	// NamespacePolicyCreate creates the namespace and errors if one with the
	// target name already exists and is not managed by this resource.
	NamespacePolicyCreate NamespacePolicy = "Create"
	// NamespacePolicyAdopt takes over a pre-existing namespace by writing the
	// management labels onto it. A namespace owned by a different Project or
	// Landscape is refused, and a missing namespace is a retryable error.
	NamespacePolicyAdopt NamespacePolicy = "Adopt"
)

// DeletionPolicy controls what happens to the managed namespace when the owning
// Project or Landscape is deleted.
type DeletionPolicy string

const (
	// DeletionPolicyDelete deletes the namespace together with the resource.
	DeletionPolicyDelete DeletionPolicy = "Delete"
	// DeletionPolicyOrphan leaves the namespace in place and only strips the
	// konfidence.cloud/* management labels.
	DeletionPolicyOrphan DeletionPolicy = "Orphan"
)
