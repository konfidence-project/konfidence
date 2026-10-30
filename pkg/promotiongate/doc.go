// Package promotiongate is the public API for writing VectorPromotion gate
// state.
//
// A VectorPromotionConfig declares gates in spec.gates, and every promotion it
// creates snapshots them into its own immutable spec.gates. Each declared gate
// has a matching entry in status.gates that carries its state (Pending,
// Satisfied, or Error) and its provenance (reason, message, setBy, setAt). A
// promotion stays Waiting until every declared gate is Satisfied. The promotion
// controller never interprets gate types; it only seeds missing entries as
// Pending and ANDs them together.
//
// # Writing a gate
//
// Every gate type has a writer, either a controller or the konfidence API, that
// observes a real-world condition and reports the result with Set:
//
//	err := promotiongate.Set(ctx, c, key, promotiongate.Update{
//	    Type:   "e2e-tests.example.com",
//	    State:  konfidence.PromotionGateStateSatisfied,
//	    Reason: "TestsPassed",
//	    SetBy:  "example.com/e2e-gate-controller",
//	})
//
// Conventions for writers:
//
//   - Name gate types <gate-name>.<owner-domain>, e.g.
//     explicit-approval.konfidence.cloud or e2e-tests.example.com. The domain
//     names the party that owns the gate type and keeps third-party gates from
//     colliding with core ones.
//   - Only act on promotions that declare your gate type in spec.gates.
//   - Report misconfiguration, such as a gate that can never pass for this
//     promotion, by setting the gate to Error with a descriptive reason and
//     message rather than leaving the promotion waiting silently.
//   - Always write through Set, so state and provenance never drift apart.
//
// A promotion's gates stay writable until they freeze. They freeze when the
// promotion controller sees every gate satisfied and sets GatesSatisfied True,
// or when the promotion finishes.
//
// Set uses optimistic locking. It reads the promotion, applies the update, and
// patches it; if another writer changed the promotion in between, Set returns
// ErrConflict and the caller retries. The apply is idempotent and upserts the
// entry, so a retry never overwrites another writer, and a write works whether
// or not the controller has seeded the entry yet.
//
// A gate controller watching VectorPromotions can filter its watch to skip work
// it cannot do. Combine Declared (promotions that carry your gate type) with
// GatesMutable (gates not yet frozen) so frozen promotions never enqueue.
//
// The recorded provenance is advisory, not a security boundary. Any actor
// allowed to patch the promotion status can write arbitrary values.
package promotiongate
