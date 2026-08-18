// Package controller implements the VectorPromotion lifecycle.
//
// # Objects and writers
//
// A `VectorPromotionConfig` declares one promotion flow (source, target
// landscape/stage) and is mutable. A `VectorPromotion` is one execution
// request against it: `spec.vector`, `spec.requireApproval` and
// `spec.sequence` are pinned at creation and immutable. Promotions are
// created by the config reconciler (vectorpromotionconfig_controller.go) when
// the source vector drifts from the target stage; it stamps `spec.sequence`
// from the config's `status.sequence` and sets a controller owner reference,
// so deleting a config garbage-collects its promotions.
//
// Conditions are the source of truth; `status.state` is display only,
// recomputed from conditions on every write. Promotion status has two
// writers — this controller and the external approver
// (`vectorpromotion.Approve`) — so every promotion status patch is
// optimistic-locked. Config status has two writers with disjoint fields —
// the execution controller mirrors `lastPromotion*`, the config reconciler
// owns `conditions` (Ready) and `sequence` — so config status patches are
// plain merge patches; that only holds while the fields stay disjoint.
//
// # Lifecycle
//
//	     (created by drift controller)
//	                |
//	requireApproval? ──no──> Ready (no Approved condition:
//	     |yes                  |    absent gates leave no record)
//	  Waiting ──Approve()──> Ready
//	                           v
//	     serialization gate (one InProgress per config,
//	     newest cleared wins, older siblings Superseded)
//	                           |
//	            target resolves? ──no──> Blocked (retried,
//	                 |yes                 config Ready=False)
//	                 v
//	            InProgress ──> Succeeded
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
// InProgress past `executionDeadline` is retired to Failed/PromotionTimedOut
// by whoever observes the overrun — itself on a retry, or a blocked sibling.
//
// # Files
//
// The reconciler is split by phase: approval
// (vectorpromotion_approval.go), serialization
// (vectorpromotion_serialization.go), execution and config mirroring
// (vectorpromotion_execution.go), condition/patch plumbing
// (vectorpromotion_status.go), TTL and retention
// (vectorpromotion_ttl_controller.go). The config reconciler is split the
// same way: reference resolution (vectorpromotionconfig_resolution.go),
// drift creation (vectorpromotionconfig_drift.go), watch mapping and
// predicates (vectorpromotionconfig_watches.go), config status plumbing
// (vectorpromotionconfig_status.go), with the reconciler and wiring in
// vectorpromotionconfig_controller.go.
package controller
