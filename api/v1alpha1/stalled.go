package v1alpha1

// StalledCondition reports that a resource cannot progress without manual intervention.
//
// Stalled uses negative polarity: Status=True means blocked and Reason carries the cause.
// A resource can stall at any point in its lifecycle.
// Controllers write it after they have evaluated the blocking conditions they own,
// False when none are present. An absent Stalled means that this evaluation has
// not completed for the resource yet.
const StalledCondition = "Stalled"

// StalledReasonNotStalled is the reason carried by Stalled=False.
const StalledReasonNotStalled = "NotStalled"
