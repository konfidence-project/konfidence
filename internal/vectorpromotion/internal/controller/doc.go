// Package controller implements the VectorPromotion lifecycle.
//
// # Objects and writers
//
// A `VectorPromotionConfig` declares one promotion flow (source, target
// landscape/stage) and is mutable. A `VectorPromotion` is one execution
// request against it: `spec.vector`, `spec.requireApproval` and
// `spec.sequence` are pinned at creation and immutable. Promotions are meant
// to be created by the drift controller (not yet implemented on this branch;
// until it lands, `spec.sequence` stays zero and ordering degrades to
// creation timestamp plus name).
//
// Conditions are the source of truth; `status.state` is display only,
// recomputed from conditions on every write. Promotion status has two
// writers — this controller and the external approver
// (`vectorpromotion.Approve`) — so every promotion status patch is
// optimistic-locked. Config status has exactly one writer, this single-worker
// controller, so config status patches are plain merge patches; that only
// holds while `MaxConcurrentReconciles` stays 1.
//
// # Lifecycle
//
//	       (created by drift controller)
//	                  |
//	  requireApproval? ──no──> Approved (AutoApproved)
//	       |yes                      |
//	WaitingForApproval ──Approve()──>|
//	                                 v
//	       serialization gate (one InProgress per config,
//	       newest approved wins, older siblings Superseded)
//	                                 |
//	                  target resolves? ──no──> Blocked (retried,
//	                       |yes                 config Ready=False)
//	                       v
//	                  InProgress ──> Succeeded
//
// Superseded, Succeeded and other Succeeded=False reasons (except Running and
// TargetUnresolved) are terminal. Terminal promotions are cleaned up by the
// TTL controller (`ttlAfterFinished`) and bounded per config by
// `keepLastPromotions`.
//
// # Crash recovery
//
// Execution is idempotent: on resume the Running patch is skipped
// (`IsInProgress`) and `promoteStage` no-ops when the stage already carries
// the vector, so a crashed reconcile simply re-runs. A promotion stuck
// InProgress stops blocking siblings after `inProgressStaleTimeout`, but is
// only driven terminal (Superseded) when a newer approved sibling executes —
// a solitary stuck promotion has no reaper and recovers only via its own
// reconcile (new events or cache resync).
//
// # Files
//
// The reconciler is split by phase: approval
// (vectorpromotion_approval.go), serialization
// (vectorpromotion_serialization.go), execution and config status
// (vectorpromotion_execution.go), condition/patch plumbing
// (vectorpromotion_status.go), TTL and retention
// (vectorpromotion_ttl_controller.go).
package controller
