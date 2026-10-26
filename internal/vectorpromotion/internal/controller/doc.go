// Package controller implements the VectorPromotion lifecycle.
//
// # Objects and writers
//
// A `VectorPromotionConfig` declares one promotion flow (source, target
// landscape/stage, gates) and is mutable. A `VectorPromotion` is one execution
// request against it: `spec.vector`, `spec.gates` and `spec.sequence` are
// pinned at creation and immutable. Promotions are
// created by the config reconciler (vectorpromotionconfig_controller.go) when
// the source vector drifts from the target stage; it stamps `spec.sequence`
// from the config's `status.sequence` and sets a controller owner reference,
// so deleting a config garbage-collects its promotions.
//
// Following the Kubernetes API conventions, a promotion has no phase field
// and its conditions are observations, not a state machine. Every decision is
// level-based on these facts and never on a condition reason:
//
//	status.gates                 per-gate state and provenance (gate writers)
//	GatesSatisfied condition     False while a gate is open (GatesPending,
//	                             GateError); latches True once all are
//	                             satisfied and is never written again
//	TargetResolved condition     whether the target Stage resolves
//	status.startTime             execution started (set once)
//	Succeeded condition          Unknown while active; True (succeeded) or
//	                             False (superseded) once finished, together
//	                             with status.completionTime; never changes
//	                             after that
//
// Reasons only explain a status. The display status of the konfidence API
// (Waiting, Ready, InProgress, Blocked, Succeeded, Superseded) is derived
// from these facts in the API layer. All writes go through `updateStatus`,
// which applies a set of mutations in one optimistic-locked status patch.
// Promotion status has several writers — this controller, and the gate
// writers (gate controllers and the konfidence API via
// `vectorpromotion.Approve`), which set gate entries through
// `pkg/promotiongate` — so a stale write conflicts and is retried on fresh
// state instead of overwriting another writer's entry. The controller never
// interprets gate types. Config status has two writers with disjoint fields —
// the config reconciler mirrors `lastPromotion*` from the promotions and owns
// `conditions` (Ready) and `sequence` — so config status patches are plain
// merge patches; that only holds while the fields stay disjoint.
//
// # Lifecycle
//
//	   (created by drift controller, gates snapshotted from the config)
//	              |
//	gate phase: seed Pending entries, Succeeded=Unknown,
//	   |        GatesSatisfied=False ──all Satisfied──> GatesSatisfied=True
//	   |   (gate in Error: reason GateError + warning event)
//	   |   (no gates: GatesSatisfied=True on the first reconcile)
//	   └── outstanding limit exceeded ──> Succeeded=False
//	        (oldest waiting promotion, reason OutstandingLimitReached)
//	                         v
//	   serialization (one started promotion per config, newest with
//	   latched gates wins, older siblings Succeeded=False/PromotionSuperseded)
//	                         |
//	          target resolves? ──no──> TargetResolved=False (retried,
//	               |yes                  config Ready=False)
//	               v
//	 TargetResolved=True, startTime ──> Stage write ──> Succeeded=True,
//	                                                    completionTime
//
// A reconcile stops for a finished promotion, runs the gate phase until
// GatesSatisfied latches and, in the same pass, continues to execution.
//
// `keepOutstandingPromotions` limits the number of promotions per
// configuration whose gates have not latched. The default limit is 10. If the
// limit is exceeded, the oldest of them, ordered by `spec.sequence`, finish
// with Succeeded=False/OutstandingLimitReached and can no longer be approved
// or executed. A value of 0 allows an unlimited number of waiting promotions.
//
// Gates can change only until GatesSatisfied latches or the promotion
// finishes (`promotiongate.Mutable`). The optimistic lock makes "all
// satisfied → latch" atomic with respect to a gate being reopened
// concurrently.
//
// Finished promotions are cleaned up by the retention controller using
// `ttlAfterFinished` (measured from `completionTime`) and bounded per config
// by `keepLastPromotions`.
//
// # Crash recovery
//
// Execution is idempotent: on resume `startTime` is already set and
// `promoteStage` no-ops when the stage already carries the vector, so a
// crashed reconcile simply re-runs. There is no execution deadline: with a
// single worker, a promotion is only observed started-but-unfinished when its
// last reconcile did not finish (crash or returned error), and its own retry
// completes it. A promotion whose Stage write is rejected for good stays
// started and keeps blocking newer promotions of its config until the cause
// is fixed.
//
// # Files
//
// The reconciler is split by phase: gates
// (vectorpromotion_gates.go), serialization
// (vectorpromotion_serialization.go), execution and config mirroring
// (vectorpromotion_execution.go), status update helpers
// (vectorpromotion_status.go), TTL and retention
// (vectorpromotion_ttl_controller.go). The config reconciler is split the
// same way: reference resolution (vectorpromotionconfig_resolution.go),
// drift creation (vectorpromotionconfig_drift.go), watch mapping and
// predicates (vectorpromotionconfig_watches.go), config status plumbing
// (vectorpromotionconfig_status.go), with the reconciler and wiring in
// vectorpromotionconfig_controller.go.
package controller
