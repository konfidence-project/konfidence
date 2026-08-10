# Async Assembly — Layer 3 Design

## Context

Layers 1 and 2 are already shipped on `perf/async-assembly`:
- **Layer 1**: `pkg/ocm/clientcache` → `pkg/lrucache` (generic LRU rename)
- **Layer 2**: `lru.Cache[string, vector.Vector]` on the reconciler — `GetVector` served from memory on cache hits, zero OCI on the no-drift hot path

This document covers **Layer 3: async assembly**. The problem it solves: even with the vector cache, `GetArtifacts` and `CreateVector` still block the controller-runtime worker goroutine for the full duration of OCM I/O. Under `MaxConcurrentReconciles=1` every other VectorTemplate waits behind the slowest OCI call.

The fix: fire OCM work on a background goroutine, return the worker to the queue immediately, apply the result on a later tick.

---

## Primitives rejected

- **`singleflight`** — adds no value under `MaxConcurrentReconciles=1`. Singleflight deduplicates concurrent callers of the same key; with serial reconciles there is only ever one caller at a time.
- **`go-pkgz/pool`** — fire-and-forget stream pool; no per-key result retrieval, no dedup, no way to hand a result to a later reconcile of the same key.

**Settled primitive**: mutex-guarded `map[jobKey]*inflightJob` on the reconciler.

---

## Data types

### Job key — generation is load-bearing

```go
type jobKey struct {
    types.NamespacedName
    generation int64
}
```

The controller-runtime work queue deduplicates by `NamespacedName`: rapid spec edits collapse to one pending entry; `r.Get` always returns the current (latest) object. The synchronous design therefore never processes a stale generation. The async design breaks this invariant:

- `gen=3` job is inflight → `gen=4` reconcile arrives
- **Without generation in key**: Reconcile sees "job running" → returns `RequeueAfter` → `gen=4` spec change silently dropped until next poll. **Correctness bug.**
- **With generation in key**: mismatch detected immediately → cancel `gen=3` context → launch `gen=4`. Spec change never lost.

### Result type — no Kubernetes types in the goroutine

```go
type assemblyOutcome int

const (
    OutcomeNoDrift assemblyOutcome = iota // HasDrift false, no OCM write
    OutcomeCreated                        // drift resolved, new vector written
    OutcomeFailed                         // OCM error
)

type assemblyResult struct {
    outcome       assemblyOutcome
    latestVector  string // OutcomeCreated: ref string written to status.latestVector
    vectorVersion string // NoDrift: current version; Created: new version (for messages)
    componentName string // for log/event messages
    err           error  // OutcomeFailed
}
```

The goroutine produces a plain Go value. All Kubernetes type construction (conditions, events) happens in the reconcile goroutine via pure mapping functions.

### Job lifecycle

```go
type jobPhase int

const (
    PhaseAssembling jobPhase = iota
    PhaseDone
    PhaseFailed
)

type jobStatus struct {
    Phase  jobPhase
    Result assemblyResult // valid when Phase is terminal
}

type inflightJob struct {
    cancel context.CancelFunc
    status chan jobStatus // buffered capacity 1; goroutine does non-blocking replace
}
```

**Channel semantics**: buffered capacity 1. The goroutine sends with a non-blocking select, replacing a stale intermediate value if one is already sitting in the buffer. This means the reconcile goroutine always reads the latest status — no goroutine leak, no blocking send.

**Methods** — channel mechanics never leak to callers:

```go
func (j *inflightJob) Status() jobStatus  // non-blocking snapshot of latest phase
func (j *inflightJob) Consume() jobStatus // drain terminal result, release resources
func (j *inflightJob) Cancel()            // cancel context; goroutine exits on next ctx check
```

---

## Reconciler additions

Two new fields, never touched outside the named methods below:

```go
type VectorTemplateReconciler struct {
    // ... existing fields ...
    mu       sync.Mutex
    inflight map[jobKey]*inflightJob
}
```

### Named methods — the only call sites for `r.mu` and `r.inflight`

```go
func (r *VectorTemplateReconciler) launchAssembly(key jobKey, ctx context.Context, ...)
func (r *VectorTemplateReconciler) pollAssembly(key jobKey) (jobStatus, bool) // false = no job for this key
func (r *VectorTemplateReconciler) completeAssembly(key jobKey)               // Consume + map delete
func (r *VectorTemplateReconciler) cancelAssembly(key jobKey)                 // Cancel + map delete
```

`launchAssembly` spawns the goroutine with a derived `context.WithCancel`, stores the `inflightJob` in the map, and returns immediately. The goroutine runs `GetArtifacts`, `HasDrift`, and optionally `CreateVector`, then pushes the result into the channel.

---

## Reconcile state machine

```
Reconcile(req):

  1. Fast sync path (always runs, cheap — no OCM I/O):
       r.Get → parse uploadTarget → resolve base VT (k8s Get) → note base latestVector ref

  2. Check inflight[{nn, gen}]:
       hit + PhaseAssembling → patch Assembling condition → return RequeueAfter(5s)
       hit + terminal        → completeAssembly(key) → apply result → patch status → return
       miss                  → continue to step 3

  3. Check for any inflight job for nn with a DIFFERENT generation:
       found → cancelAssembly(oldKey)   (context cancelled, goroutine exits at next ctx check)

  4. Vector cache + drift check (cheap — memory only if both vectors cached):
       own + base cache hits AND HasDrift false → NoDriftDetected condition → return
       cache miss OR drift detected             → launchAssembly(key, ...) → patch Assembling
                                                → return RequeueAfter(5s)
```

**What stays synchronous**: the `r.Get` on the base VectorTemplate (step 1). Base resolution needs the current cluster state at dispatch time. The resolved base artifacts are passed *into* the goroutine as a value — the goroutine never touches the Kubernetes API.

---

## Correctness: defence in depth

Three independent layers, each catching a different failure mode:

| Layer | Mechanism | What it catches |
|-------|-----------|-----------------|
| 1 | `jobKey` includes `generation` | Stale inflight job detected at step 3; `gen=3` cancelled before `gen=4` launches |
| 2 | Apply-time `vt.Generation == job.key.generation` guard before status patch | `gen=3` goroutine finishes *after* `gen=4` already wrote status; stale result silently discarded |
| 3 | `resourceVersion` optimistic lock via `client.MergeFrom` | API server returns 409 on any conflicting concurrent write → controller-runtime backoff → requeue → fresh `r.Get` → self-healing |

Layer 3 is a free backstop: `client.MergeFrom` captures `resourceVersion` at `r.Get` time. Any concurrent write (spec change, another controller's status patch) bumps `resourceVersion`, causing the patch to fail with 409. The backoff requeue picks up the latest object and self-heals. Note: `resourceVersion` increments on **any** write (spec or status subresource), not just spec changes — so it catches more races than `generation` alone.

---

## Status surface — pure mapping functions

Kubernetes types are never constructed inside the goroutine. Three pure functions handle the translation:

```go
// assemblyResultToCondition maps a terminal result to the Ready condition.
func assemblyResultToCondition(result assemblyResult, vt *konfidence.VectorTemplate) metav1.Condition

// assemblyResultToEvent extracts event fields from a terminal result.
func assemblyResultToEvent(result assemblyResult) (eventType, reason, action, message string)

// assemblingCondition produces the in-progress condition patched while the goroutine runs.
func assemblingCondition(vt *konfidence.VectorTemplate) metav1.Condition
```

### Observable status surface (non-breaking)

| Outcome | Condition reason | Message |
|---------|-----------------|---------|
| NoDrift | `NoDriftDetected` | `"No drift detected for vector - vector version is still %s"` |
| Created | `VectorCreated` | `"Drift detected and new vector created successfully - new vector version is %s"` |
| Failed  | `VectorCreationFailed` | `err.Error()` |
| In-progress (new) | `Assembling` | `"Assembling vector in background"` |

`Assembling` is the only new observable state. It is transient: the next reconcile tick (RequeueAfter 5s) replaces it with a terminal condition.

---

## Test additions

Existing two-phase drift specs (`Eventually` polls) already tolerate async behaviour — no structural changes needed.

Add two new specs:

1. **`Assembling` condition appears before `VectorCreated`**: trigger drift, assert `Ready=False/Assembling` within the first requeue window, then assert `Ready=True/VectorCreated` after the goroutine completes.

2. **Generation guard**: create a VectorTemplate, trigger a spec change while a slow assembly is inflight (use a mock adapter with a delay), assert the stale result is not applied — `status.latestVector` reflects the `gen=N+1` outcome, not the cancelled `gen=N`.

---

## File map

All changes are in `internal/vectorassembly/internal/controller/`:

| File | Change |
|------|--------|
| `vectortemplate_controller.go` | Add `mu`, `inflight` fields; add `jobKey`, `assemblyOutcome`, `assemblyResult`, `jobPhase`, `jobStatus`, `inflightJob` types; add `launchAssembly`, `pollAssembly`, `completeAssembly`, `cancelAssembly` methods; rework `Reconcile` to the state machine above; add `assemblyResultToCondition`, `assemblyResultToEvent`, `assemblingCondition` |
| `vectortemplate_controller_test.go` | Add `Assembling` condition spec and generation guard spec |

No API type changes. No changes outside the vectorassembly controller package.