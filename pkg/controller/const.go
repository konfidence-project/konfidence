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
	// VectorDeploymentNameLabel records the name of the owning VectorDeployment.
	VectorDeploymentNameLabel = "konfidence.cloud/vector-deployment-name"
	// VectorDeploymentUIDLabel records the UID of the owning VectorDeployment, useful
	// when the deployment is recreated under the same name.
	VectorDeploymentUIDLabel = "konfidence.cloud/vector-deployment-uid"
)
