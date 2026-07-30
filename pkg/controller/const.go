package utils

const (
	StageNameLabel          = "konfidence.cloud/stage-name"
	StageVersionNameLabel   = "konfidence.cloud/stage-version-name"
	VectorReferenceLabel    = "konfidence.cloud/vector-ref"
	StageVersionUsageTarget = "konfidence.cloud/target-for-stage"
	ArtifactReferenceLabel  = "konfidence.cloud/artifact-ref"

	// ManagedByLabel marks objects materialized by a Konfidence controller. Its value
	// is the name of the controller (e.g. "vector-deployment-controller").
	ManagedByLabel = "konfidence.cloud/managed-by"
	// ProjectTypeLabel marks the kind of a Konfidence-managed namespace
	// (value "project" on project namespaces).
	ProjectTypeLabel = "konfidence.cloud/type"
	// ProjectNameLabel records the name of the owning Project.
	ProjectNameLabel = "konfidence.cloud/project"
	// VectorDeploymentNameLabel records the name of the owning VectorDeployment.
	VectorDeploymentNameLabel = "konfidence.cloud/vector-deployment-name"
	// VectorDeploymentUIDLabel records the UID of the owning VectorDeployment, useful
	// when the deployment is recreated under the same name.
	VectorDeploymentUIDLabel = "konfidence.cloud/vector-deployment-uid"

	// ArtifactComponentAnnotation records the artifact component name on an ArtifactDeployment.
	ArtifactComponentAnnotation = "konfidence.cloud/artifact-component"
	// ArtifactVersionAnnotation records the artifact version on an ArtifactDeployment.
	ArtifactVersionAnnotation = "konfidence.cloud/artifact-version"
	// ArtifactHashAnnotation records the hash used to derive the ArtifactDeployment name.
	ArtifactHashAnnotation = "konfidence.cloud/artifact-hash"
	// VectorDeploymentUIDAnnotation records the UID of the vector deployment that owns this artifact.
	VectorDeploymentUIDAnnotation = "konfidence.cloud/vector-deployment-uid"
	// AllowReuseAnnotation marks an ArtifactDeployment as reusable across vector deployments.
	AllowReuseAnnotation = "konfidence.cloud/allow-reuse"
)
