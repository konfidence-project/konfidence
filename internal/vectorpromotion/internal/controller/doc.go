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
// The promotion phase is the reason of the `Succeeded` condition; its status
// tells whether the promotion finished (`Unknown` while live, `True` on
// success, `False` on any other end). One table in the `promotion` package
// pairs every reason with its condition status and its state; `DeriveState`
// classifies a promotion and every decision compares that state. The phase
// is written only through `transition`, which looks up the condition status,
// stamps `lastTransitionTime` on a reason change and writes `status.state`
// in the same optimistic-locked patch, so the two never disagree.
// `status.gates` is input to the gate phase only. Promotion status has several writers — this
// controller, which seeds missing gate entries as `Pending` and writes the
// condition, and the gate writers (gate controllers and the konfidence API
// via `vectorpromotion.Approve`), which set gate entries through
// `pkg/promotiongate` — so every promotion status patch is optimistic-locked:
// a stale write conflicts and is retried on fresh state instead of
// overwriting another writer's entry. The controller never interprets gate
// types. Config status has two writers with disjoint fields —
// the config reconciler mirrors `lastPromotion*` from the promotions and owns
// `conditions` (Ready) and `sequence` — so config status patches are
// plain merge patches; that only holds while the fields stay disjoint.
//
// # Lifecycle
//
//	Succeeded reason (status)          display state
//	───────────────────────────────    ─────────────
//	(none yet)                         Waiting
//	GatesPending / GateError (Unknown) Waiting      ← gate phase
//	PromotionReady           (Unknown) Ready        ← gates frozen from here
//	PromotionRunning         (Unknown) InProgress
//	PromotionTargetUnresolved(Unknown) Blocked
//	PromotionSucceeded       (True)    Succeeded
//	PromotionSuperseded /
//	  OutstandingLimitReached (False)  Superseded
//	PromotionFailed          (False)   Failed
//
//	     (created by drift controller, gates snapshotted from the config)
//	                |
//	  gate phase: seed Pending entries ──all Satisfied──> PromotionReady
//	     |   (gate in Error: GateError + warning event, stays waiting)
//	     |   (no gates: PromotionReady on the first reconcile)
//	     └── outstanding limit exceeded ──> Superseded
//	          (oldest waiting promotion, reason OutstandingLimitReached)
//	                           v
//	     serialization (one InProgress per config,
//	     newest Ready/InProgress/Blocked wins, older siblings Superseded)
//	                           |
//	            target resolves? ──no──> Blocked (retried,
//	                 |yes                 config Ready=False)
//	                 v
//	            InProgress ──> Succeeded
//
// A reconcile runs the gate phase while the state is Waiting and, if that
// records PromotionReady, continues to execution in the same pass. A
// promotion competes for execution while its state is Ready, InProgress or
// Blocked (`VectorPromotionState.Cleared`); the gates are not re-read.
//
// `keepOutstandingPromotions` limits the number of promotions in the `Waiting`
// state per configuration. The default limit is 10. If the limit is exceeded,
// the oldest waiting promotions, ordered by `spec.sequence`, transition to
// `Superseded` with reason `OutstandingLimitReached` and can no longer be
// approved or executed. A value of 0 allows an unlimited number of waiting
// promotions. Promotions in any other state do not count toward this limit.
//
// Gates can change only while the state is Waiting (`pkg/promotiongate`
// reads `status.state`). Recording PromotionReady freezes them, and the
// optimistic lock makes "all satisfied → Ready" atomic with respect to a
// gate being reopened concurrently.
//
// Succeeded, Failed and Superseded are terminal
// (`VectorPromotionState.Terminal`). Terminal
// promotions are cleaned up by the retention controller using
// `ttlAfterFinished` and bounded per config by `keepLastPromotions`. Every
// reason change stamps `lastTransitionTime`, so it always marks when the
// current phase began.
//
// # Crash recovery
//
// Execution is idempotent: on resume the Running patch is skipped (state
// already InProgress) and `promoteStage` no-ops when the stage already carries
// the vector, so a crashed reconcile simply re-runs. There is no execution
// deadline: with a single worker, a promotion is only observed InProgress
// when its last reconcile did not finish (crash or returned error), and its
// own retry completes it. A promotion whose Stage write is rejected for good
// stays InProgress and keeps blocking newer promotions of its config until
// the cause is fixed.
//
// # Files
//
// The reconciler is split by phase: gates
// (vectorpromotion_gates.go), serialization
// (vectorpromotion_serialization.go), execution and config mirroring
// (vectorpromotion_execution.go), the transition helper
// (vectorpromotion_status.go), TTL and retention
// (vectorpromotion_ttl_controller.go). The config reconciler is split the
// same way: reference resolution (vectorpromotionconfig_resolution.go),
// drift creation (vectorpromotionconfig_drift.go), watch mapping and
// predicates (vectorpromotionconfig_watches.go), config status plumbing
// (vectorpromotionconfig_status.go), with the reconciler and wiring in
// vectorpromotionconfig_controller.go.
package controller
