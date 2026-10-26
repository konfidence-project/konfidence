// Package promotiongate is the public contract for writing VectorPromotion
// gate state (ADR-0039).
//
// A VectorPromotionConfig declares gates in `spec.gates`; every promotion it
// creates snapshots them into its own immutable `spec.gates`. Each declared
// gate has a matching entry in `status.gates` carrying its state (`Pending`,
// `Satisfied`, `Error`) together with its provenance (`reason`, `message`,
// `setBy`, `setAt`). The promotion stays `Waiting` until every declared gate is
// `Satisfied`. The promotion controller never interprets gate types; it only
// seeds missing entries as `Pending` and ANDs them together.
//
// # Writing a gate
//
// Every gate type has a writer, either a controller or the konfidence API,
// that observes a real-world condition and reports the result with Set:
//
//	err := promotiongate.Set(ctx, c, key, promotiongate.Update{
//		Type:   "e2e-tests.example.com",
//		State:  konfidence.PromotionGateStateSatisfied,
//		Reason: "TestsPassed",
//		SetBy:  "example.com/e2e-gate-controller",
//	})
//
// Conventions for writers:
//
//   - Gate types are named `<gate-name>.<owner-domain>`, e.g.
//     `explicit-approval.konfidence.cloud` or `e2e-tests.example.com`. The
//     domain names the party that owns the gate type and keeps third-party
//     gates from colliding with core ones.
//   - Only act on promotions that declare your gate type in `spec.gates`.
//   - Report misconfiguration (e.g. a gate that can never pass for this
//     promotion) by setting the gate to `Error` with a descriptive reason and
//     message rather than leaving the promotion waiting silently.
//   - Always write through Set, so state and provenance never drift apart.
//
// Gates can only change while the promotion has not started executing, i.e.
// while it is `Waiting` or `Ready`. Set writes are optimistic-locked and
// retried on conflict, so concurrent writers (other gates, the promotion
// controller seeding entries) never overwrite each other. Set upserts: it does
// not depend on the promotion controller having seeded the entry first.
//
// The recorded provenance is advisory and not a security boundary: any actor
// allowed to patch the promotion status can write arbitrary values.
package promotiongate
