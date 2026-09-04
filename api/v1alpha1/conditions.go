package v1alpha1

// StalledCondition reports that a resource has reached a state it cannot leave without
// manual intervention. It is shared by VectorDeployment and ArtifactDeployment.
//
// Unlike the lifecycle conditions, which are positive-polarity and report progress
// ("did this step succeed yet"), Stalled is abnormal-true: Status=True means blocked.
// It is orthogonal to the lifecycle conditions rather than a stage within them, so a
// resource can be mid-pipeline and stalled at the same time.
//
// Controllers MUST write Stalled on every reconcile, setting it to False when nothing
// blocks. Tooling that follows the kstatus convention reads an absent Stalled as "not
// stalled" rather than as Unknown, so omitting it on the healthy path would make "no
// blocking cause found" indistinguishable from "never evaluated". With it always
// written, an absent Stalled means only that the object has never been reconciled.
//
// The blocking cause is carried in the condition's Reason, not in separate condition
// types, so that "is this blocked?" stays a single orthogonal axis. Only one reason can
// be reported at a time; see the precedence rule below.
const StalledCondition = "Stalled"

// Stalled reasons written by the VectorDeployment controller.
const (
	// StalledReasonArtifactDeploymentNamingCollision indicates that a deterministic
	// ArtifactDeployment name kept colliding with a different artifact after the
	// collisionCount salt was exhausted. Ordinary collisions are self-healing (the salt
	// is bumped and the next reconcile deploys under a fresh name), so this reason is
	// only set once bumping has stopped helping, which indicates a bug or a hash-space
	// problem that a human has to resolve.
	StalledReasonArtifactDeploymentNamingCollision = "ArtifactDeploymentNamingCollision"

	// StalledReasonChildArtifactDeploymentStalled indicates that at least one
	// ArtifactDeployment belonging to this vector is itself stalled. It is a derived
	// cause: the VectorDeployment is not blocked by its own reconcile logic but cannot
	// reach Ready while a child is blocked. The child's name and its own reason are
	// carried in the message.
	StalledReasonChildArtifactDeploymentStalled = "ChildArtifactDeploymentStalled"
)

// Stalled reasons written by the deployer that owns an ArtifactDeployment. The deployer
// is runtime-specific and lives outside this repository; these constants exist so that
// deployers, the API layer and tests bind to one contract rather than to string
// literals. Nothing in this repository sets them.
const (
	// StalledReasonManifestMissing indicates that the Konfidence manifest is absent from
	// the artifact. A fetch that failed transiently is not this reason.
	StalledReasonManifestMissing = "ManifestMissing"

	// StalledReasonNoDeploymentClassAvailable indicates that no DeploymentClass matches
	// the artifact, so no deployer can claim it.
	StalledReasonNoDeploymentClassAvailable = "NoDeploymentClassAvailable"

	// StalledReasonDeploymentTargetSecretsMissing indicates that credentials required by
	// the DeploymentTarget are absent. Secrets are often delivered asynchronously by an
	// external syncer, so this reason is deadline-based: it should be set only after the
	// secrets have stayed missing past a threshold, not on the first reconcile that
	// observes them missing.
	StalledReasonDeploymentTargetSecretsMissing = "DeploymentTargetSecretsMissing"

	// StalledReasonDeploymentResultNotUnique indicates that the deployer produced two
	// DeploymentResults sharing a (name, type) pair, which the API requires to be unique.
	StalledReasonDeploymentResultNotUnique = "DeploymentResultNotUnique"
)

// Stalled reason precedence, when more than one cause holds at once.
//
// Only one reason can be reported, so the choice has to be deterministic: picking
// whichever check happens to run first makes the condition flap as unrelated code moves
// around, and makes tests order-dependent. The rule is:
//
//	Terminal causes always outrank deadline-elapsed ones. Within the same class, the
//	cause blocking the earlier pipeline stage wins. A derived cause takes the class of
//	the cause it derives from.
//
// Terminal causes are decidable immediately: retrying cannot change the answer, only a
// human editing spec or cluster state can. Deadline-elapsed causes are those where
// "missing now" may still mean "not yet", and which therefore only become blocking after
// a threshold. The third clause covers ChildArtifactDeploymentStalled, which is neither
// on its own and inherits whichever class the child's own reason belongs to.
//
// The rule matters where it disagrees with pipeline order alone: a deployer seeing both
// DeploymentTargetSecretsMissing (deadline-elapsed, earlier stage) and
// DeploymentResultNotUnique (terminal, later stage) reports the terminal one.
