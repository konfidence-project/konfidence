// Package controller implements the VectorPromotion lifecycle.
//
// # Objects
//
// A VectorPromotionConfig declares one promotion flow (source, target
// landscape/stage, and gates) and is mutable. A VectorPromotion is a single
// execution request against that config: its spec.vector, spec.gates, and
// spec.sequence are pinned at creation and immutable.
//
// The config reconciler (vectorpromotionconfig_controller.go) creates a
// VectorPromotion whenever the source vector drifts from the target stage. It
// sets a controller owner reference, so deleting a config garbage-collects its
// promotions.
//
// # Status and writers
//
// Following the Kubernetes API conventions, a promotion has no phase field;
// its conditions are observations, not a state machine.
//
//	status.gates                 per-gate state and provenance (gate writers)
//	status.override              operator decision (Reject or BypassGates) and
//	                             its provenance (konfidence API); set at most
//	                             once while waiting, never cleared
//	GatesSatisfied condition     False while any gate is unsatisfied
//	                             (GatesPending, GateError) or an operator
//	                             bypassed the gates (GatesBypassed); becomes
//	                             True once every gate is satisfied, then is
//	                             never written again
//	TargetResolved condition     whether the target Stage resolves
//	status.startTime             execution started (set once)
//	Succeeded condition          Unknown while active; True (succeeded) or
//	                             False (superseded or rejected) once
//	                             finished, together with
//	                             status.completionTime; never changes after
//	                             that
//
// Reasons only explain a status. The API layer derives the konfidence display
// status (Waiting, Ready, InProgress, Blocked, Succeeded, Superseded,
// Rejected) from these facts and the override.
//
// Promotion status has several writers: this controller, the gate writers
// (gate controllers and the konfidence API via vectorpromotion.Approve) that
// set gate entries through pkg/promotiongate, and the konfidence API that
// records the operator override (vectorpromotion.Reject and
// vectorpromotion.BypassGates). Status writes use optimistic locking to handle
// conflicts. The controller never interprets gate types.
// Every lifecycle reconciliation writes only its own promotion's status;
// sibling changes enqueue observations, never transitions on other objects.
//
// # Lifecycle
//
//	a VectorPromotion is created by the drift controller;
//	its gates are copied from the config and frozen at creation
//	  │
//	  ▼
//	┌──────────────────────────────────────────────────┐
//	│ GATE PHASE                                       │
//	│   seed a Pending entry per declared gate         │
//	│   Succeeded=Unknown, GatesSatisfied=False        │
//	└──────────────────────────────────────────────────┘
//	  │
//	  ├──▶ a gate errors → GateError reason + warning event (keeps waiting)
//	  │
//	  ├──▶ override Reject → Succeeded=False
//	  │        (PromotionRejected, terminal; takes precedence over the gates)
//	  │
//	  ├──▶ outstanding limit exceeded → Succeeded=False
//	  │        (OutstandingLimitReached, terminal)
//	  │
//	  ▼   all gates satisfied (immediately when there are none) → GatesSatisfied=True;
//	  │   or override BypassGates → GatesSatisfied=False/GatesBypassed, gate entries untouched.
//	  │   Either way the gates freeze and the promotion is ready for execution.
//	  │
//	  ▼
//	┌──────────────────────────────────────────────────┐
//	│ SERIALIZATION                                    │
//	│   one started promotion per config               │
//	│   newest ready for execution wins                │
//	└──────────────────────────────────────────────────┘
//	  │
//	  ├──▶ a newer ready or terminal, non-rejected sibling exists
//	  │        → Succeeded=False (PromotionSuperseded, terminal)
//	  │
//	  ▼
//	target Stage resolves?
//	  │
//	  ├──▶ no → TargetResolved=False, retried (config Ready=False)
//	  │
//	  ▼   yes
//	TargetResolved=True, startTime set
//	  │
//	  ▼
//	┌──────────────────────────────────────────────────┐
//	│ EXECUTION                                        │
//	│   write the promoted vector to the Stage         │
//	└──────────────────────────────────────────────────┘
//	  │
//	  ▼
//	Succeeded=True, completionTime set
//
// keepOutstandingPromotions limits how many promotions per config may have
// unsatisfied gates (default 10). When the limit is exceeded, the oldest of
// them, ordered by spec.sequence, finish with
// Succeeded=False/OutstandingLimitReached and can no longer be approved or
// executed. A value of 0 allows unlimited waiting promotions.
// The lifecycle reconciler applies this bound to itself by counting newer
// waiting siblings; the TTL controller never writes lifecycle status.
//
// Waiting and ready promotions both supersede themselves when a newer sibling
// is ready for execution (satisfied gates or bypass) or terminal (Succeeded=True
// or False). A newer ready promotion need not have started execution yet; a
// failed promotion also prevents older execution. Pending gates or gate errors
// alone do not supersede older promotions. Terminal promotions never change.
// Rejected promotions never supersede another promotion, even if their gates
// were cleared before the rejection or the rejection is not yet reconciled.
// Supersession is irreversible: a later rejection or failure of the newer
// promotion does not revive older terminal promotions. Recovery needs a new
// promotion.
//
// # Sibling observations and recovery
//
// A second VectorPromotion watch maps create/delete events and changes to
// supersession eligibility, execution start, or terminal state to non-terminal
// siblings of the same config in the same project namespace. Both older siblings (supersession
// and outstanding limits) and newer siblings (execution unblocking) are queued.
// Config generation changes enqueue active promotions to reapply outstanding
// limits. Gate provenance and condition-message-only updates do not fan out.
//
// Every enqueued resource decides from the default informer cache. A stale
// sibling observation can delay supersession, so execution has a separate guard:
// a Stage records the highest executed sequence per config incarnation in
// konfidence.cloud/promotion-sequences. The sequence and vector are written in
// one optimistic-locked patch. If the cached Stage is stale, the patch conflicts;
// when the cache catches up, an older promotion sees the watermark and marks
// itself superseded without writing the Stage. No uncached reads, in-memory
// expectations, or additional reconciliation phases are needed.
//
// The watermark survives promotion retention and controller restarts. Each
// config has its own entry so flows targeting the same Stage do not erase each
// other's execution history. Config owner UIDs distinguish recreated configs;
// unowned, manually created promotions fall back to namespace/config-name.
// The annotation must be preserved; a deleted/recreated Stage starts fresh.
// This fences writes to the same target Stage; different target Stages do not
// share a watermark. Sibling status observations still converge via watches.
// Status writes also use optimistic locking, and the lifecycle controller keeps
// a single worker with leader election across multiple replicas.
// Non-terminal promotions periodically retry as a fallback for failed watch
// mapping. Initial informer events after a restart also drive convergence.
//
// Gates can change only until GatesSatisfied is set, an override is recorded,
// or the promotion finishes. The optimistic lock makes "all satisfied →
// GatesSatisfied set" atomic with respect to a gate being reopened or an
// override being recorded concurrently.
//
// The retention controller cleans up finished promotions using ttlAfterFinished
// (measured from completionTime) and bounds them per config by
// keepLastPromotions.
// Completed promotions that are still needed as supersession evidence by an
// older active promotion are retained temporarily, even beyond TTL or count
// limits. Once older promotions reconcile to terminal states, cleanup resumes.
//
// # Files
//
// The VectorPromotion reconciler spans several files:
//
//   - vectorpromotion_gates.go: gate seeding and the GatesSatisfied condition
//   - vectorpromotion_status.go: status update helpers
//   - vectorpromotion_controller.go: lifecycle entry point and watch mapping
//   - vectorpromotion_serialization.go: execution eligibility, self-supersession, outstanding limits
//   - vectorpromotion_execution.go: target resolution and Stage write
//   - vectorpromotion_ttl_controller.go: TTL, retention, and supersession evidence protection
//
// The VectorPromotionConfig reconciler spans several files too:
//
//   - vectorpromotionconfig_resolution.go: reference resolution
//   - vectorpromotionconfig_drift.go: drift detection and promotion creation
//   - vectorpromotionconfig_watches.go: watch mapping and predicates
//   - vectorpromotionconfig_status.go: config status plumbing
//   - vectorpromotionconfig_controller.go: reconciler and wiring
package controller
