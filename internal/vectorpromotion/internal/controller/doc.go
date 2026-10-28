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
//	GatesSatisfied condition     False while any gate is unsatisfied
//	                             (GatesPending, GateError); becomes True once
//	                             all gates are satisfied, then is never written
//	                             again
//	TargetResolved condition     whether the target Stage resolves
//	status.startTime             execution started (set once)
//	Succeeded condition          Unknown while active; True (succeeded) or
//	                             False (superseded) once finished, together
//	                             with status.completionTime; never changes
//	                             after that
//
// Reasons only explain a status. The API layer derives the konfidence display
// status (Waiting, Ready, InProgress, Blocked, Succeeded, Superseded) from
// these facts.
//
// Promotion status has several writers: this controller, and the gate writers
// (gate controllers and the konfidence API via vectorpromotion.Approve) that
// set gate entries through pkg/promotiongate. Status writes use optimistic
// locking to handle conflicts. The controller never interprets gate types.
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
//	  ├──▶ outstanding limit exceeded → Succeeded=False
//	  │        (OutstandingLimitReached, terminal)
//	  │
//	  ▼   all gates satisfied (immediately when there are none)
//	GatesSatisfied=True → gates freeze
//	  │
//	  ▼
//	┌──────────────────────────────────────────────────┐
//	│ SERIALIZATION                                    │
//	│   one started promotion per config               │
//	│   newest with satisfied gates wins               │
//	└──────────────────────────────────────────────────┘
//	  │
//	  ├──▶ older siblings → Succeeded=False
//	  │        (PromotionSuperseded, terminal)
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
//
// Gates can change only until GatesSatisfied is set or the promotion finishes.
// The optimistic lock makes "all satisfied → GatesSatisfied set" atomic with
// respect to a gate being reopened concurrently.
//
// The retention controller cleans up finished promotions using ttlAfterFinished
// (measured from completionTime) and bounds them per config by
// keepLastPromotions.
//
// # Files
//
// The VectorPromotion reconciler spans several files:
//
//   - vectorpromotion_gates.go: gate seeding and the GatesSatisfied condition
//   - vectorpromotion_serialization.go: serialization and superseding siblings
//   - vectorpromotion_execution.go: target resolution, Stage write, and config mirroring
//   - vectorpromotion_status.go: status update helpers
//   - vectorpromotion_ttl_controller.go: TTL and retention
//
// The VectorPromotionConfig reconciler spans several files too:
//
//   - vectorpromotionconfig_resolution.go: reference resolution
//   - vectorpromotionconfig_drift.go: drift detection and promotion creation
//   - vectorpromotionconfig_watches.go: watch mapping and predicates
//   - vectorpromotionconfig_status.go: config status plumbing
//   - vectorpromotionconfig_controller.go: reconciler and wiring
package controller
