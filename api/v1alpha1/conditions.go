package v1alpha1

// StalledCondition reports that a resource cannot progress without manual intervention.
// Shared by VectorDeployment and ArtifactDeployment.
//
// Abnormal-true: Status=True means blocked. Orthogonal to the lifecycle conditions, so a
// resource can be mid-pipeline and stalled at once. The blocking cause is carried in Reason.
//
// Controllers must write it on every reconcile, False when nothing blocks, so that an
// absent Stalled means only that the object has never been reconciled.
const StalledCondition = "Stalled"

// Stalled reasons written by the VectorDeployment controller.
const (
	// StalledReasonArtifactDeploymentNamingCollision: a deterministic ArtifactDeployment
	// name still collides after the collisionCount salt is exhausted. Ordinary collisions
	// self-heal by bumping the salt.
	StalledReasonArtifactDeploymentNamingCollision = "ArtifactDeploymentNamingCollision"

	// StalledReasonChildArtifactDeploymentStalled: an ArtifactDeployment of this vector is
	// itself stalled. Child name and reason are in the message.
	StalledReasonChildArtifactDeploymentStalled = "ChildArtifactDeploymentStalled"
)

// Stalled reasons written by the runtime-specific deployer that owns an ArtifactDeployment.
// Declared here so both sides bind to one contract. Nothing in this repository sets them.
const (
	// StalledReasonManifestMissing: the Konfidence manifest is absent from the artifact.
	// A transient fetch failure is not this reason.
	StalledReasonManifestMissing = "ManifestMissing"

	// StalledReasonNoDeploymentClassAvailable: no DeploymentClass matches the artifact.
	StalledReasonNoDeploymentClassAvailable = "NoDeploymentClassAvailable"

	// StalledReasonDeploymentTargetSecretsMissing: DeploymentTarget credentials are absent.
	// Deadline-based, since secrets often arrive asynchronously.
	StalledReasonDeploymentTargetSecretsMissing = "DeploymentTargetSecretsMissing"

	// StalledReasonDeploymentResultNotUnique: two DeploymentResults share a (name, type).
	StalledReasonDeploymentResultNotUnique = "DeploymentResultNotUnique"
)
