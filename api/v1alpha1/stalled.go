package v1alpha1

// StalledCondition reports that a resource cannot progress without manual intervention.
//
// Status=True means blocked and Reason carries the cause. It is independent of Ready, so a
// resource can be both. Controllers write False when nothing blocks; absent means not yet evaluated.
const StalledCondition = "Stalled"

// StalledReasonNotStalled is the reason carried by Stalled=False.
const StalledReasonNotStalled = "NotStalled"

// StalledReasonStalled is the reason for generic stall cases.
const StalledReasonStalled = "Stalled"
