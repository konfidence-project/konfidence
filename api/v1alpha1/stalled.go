package v1alpha1

// StalledCondition reports that a resource cannot progress without manual intervention.
// Shared by VectorDeployment and ArtifactDeployment.
//
// Abnormal-true: Status=True means blocked and Reason carries the cause. A resource can stall
// at any point in its lifecycle. Stalled=True implies Ready=False.
//
// Controllers write it once they have evaluated what could block, False when nothing does.
// An absent Stalled means that evaluation has not happened yet.
const StalledCondition = "Stalled"

// StalledReasonNotStalled is the reason carried by Stalled=False.
const StalledReasonNotStalled = "NotStalled"
