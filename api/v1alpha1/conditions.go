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

// StalledReasonNotStalled is the reason carried by Stalled=False.
const StalledReasonNotStalled = "NotStalled"
