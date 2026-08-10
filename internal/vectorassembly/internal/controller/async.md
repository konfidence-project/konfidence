# Async Assembly — Layer 3 Design

## Context

Layers 1 and 2 are already shipped on `perf/async-assembly`:
- **Layer 1**: `lrucache.Cache` for OCM adapters — caches credentials resolver + OCI client + verifiers by `namespace/name/generation`. A spec change bumps generation → natural eviction.
- **Layer 2**: `lru.Cache[string, vector.Vector]` for fetched vectors — `getVectorCached` serves from memory on ref hits, avoiding OCI reads for vectors that haven't changed.

This document covers **Layer 3: async assembly**. Even with both caches, `GetArtifacts` (OCI reads for upstream component descriptors) and `CreateVector` (OCI write) block the controller-runtime worker goroutine. Under `MaxConcurrentReconciles=1` every other VectorTemplate waits behind the slowest OCI call.

The fix: fire OCM work on a background goroutine, return the worker to the queue immediately, apply the result on a later tick.

---

## Why there is no synchronous drift-check path

Drift detection compares a `currentVector` (what we last built) against a `desiredVector` (what we want now). The desired vector is constructed from `GetArtifacts` results — upstream component descriptors fetched from OCI. This call always hits the network because its purpose is to detect upstream version changes that produce no Kubernetes event.

Therefore every reconcile that wants to answer "is there drift?" must do at least one OCI call. The async design accepts this and launches a background job on every reconcile cycle.

---

## Primitive

Mutex-guarded `map[types.NamespacedName]*inflightJob` on the reconciler.

Rejected alternatives:
- **`singleflight`** — deduplicates concurrent callers of the same key; useless under `MaxConcurrentReconciles=1` where there is only ever one caller.
- **`go-pkgz/pool`** — fire-and-forget; no per-key result retrieval, no way to hand a result to a later reconcile.

---

## Types

```go
type assemblyResult struct {
    latestVector  string // non-empty when drift was resolved and a new vector was written
    vectorVersion string // current (no-drift) or new (created) version for condition messages
    componentName string // for log/event messages
    err           error  // non-nil means the assembly failed
}
```

Interpreting the result:
- `err != nil` → assembly failed (OCI error)
- `err == nil && latestVector != ""` → drift resolved, new vector written
- `err == nil && latestVector == ""` → no drift detected

```go
type inflightJob struct {
    generation int64
    cancel     context.CancelFunc
    result     chan assemblyResult // buffered cap 1, sent exactly once
}
```

A job is **done** when `len(job.result) > 0`. No lifecycle enum — the result encodes the outcome.

---

## Reconciler

```go
type VectorTemplateReconciler struct {
    // ... existing fields ...
    mu       sync.Mutex
    inflight map[types.NamespacedName]*inflightJob
}
```

Helper methods (the only code that touches `r.mu`/`r.inflight`):

```go
func (r *Reconciler) getJob(nn types.NamespacedName) (*inflightJob, bool)
func (r *Reconciler) launchJob(nn types.NamespacedName, gen int64, fn func() assemblyResult) *inflightJob
func (r *Reconciler) removeJob(nn types.NamespacedName)
```

`launchJob` creates a `context.WithCancel`, stores the job, and spawns a goroutine that calls `fn()` and sends the result into the channel.

---

## What the reconcile goroutine does (synchronous — no OCI)

Before checking or launching a job, the reconcile goroutine performs cheap Kubernetes-only setup:

1. `r.Get(ctx, req.NamespacedName, vt)` — fetch the VectorTemplate
2. `r.Cache.Lookup(ctx, r.Client, vt)` — get or create the OCM adapter (Layer 1 cache; resolves credentials from k8s Secrets on miss)
3. Parse `spec.components` → `[]compref.Ref` and `spec.uploadTarget` → upload ref
4. If `spec.base != nil`: `r.Get` the base VectorTemplate → read `status.latestVector` → if empty, patch "waiting for base" condition and return (no requeue — re-enqueue comes from the base VT watch)

All values computed here are passed **by value** into the background goroutine. The goroutine never calls `r.Get` or touches Kubernetes types.

---

## What the background goroutine does (OCI I/O)

The goroutine receives: the OCM adapter, component refs, upload target ref, base vector ref (if applicable), current vector ref (from `status.latestVector`), desired vector config, and the version generator.

```
1. If base vector ref is set:
     baseVector := getVectorCached(adapter, baseRef)   // Layer 2 cache hit or OCI read
     baseArtifacts := baseVector.Artifacts

2. If current vector ref is set:
     currentVector := getVectorCached(adapter, currentRef) // Layer 2 cache hit or OCI read
   else:
     currentVector is zero (first build → drift will be detected)

3. componentArtifacts := adapter.GetArtifacts(ctx, componentRefs)  // ALWAYS OCI — fetches upstream descriptors

4. desiredArtifacts := combine(baseArtifacts, componentArtifacts)  // base artifacts overwritten by same-name component artifacts
   desiredVector := Vector{Artifacts: desiredArtifacts, VectorConfig: vectorConfig}

5. if !HasDrift(currentVector, desiredVector):
     return assemblyResult{vectorVersion: currentVector.Version}   // no drift

6. newVersion := versionGenerator.Generate()
   newVector := desiredVector with version = newVersion
   adapter.CreateVector(ctx, uploadTarget, newVector)              // OCI write
   return assemblyResult{latestVector: refString, vectorVersion: newVersion, componentName: ...}
```

On any error at steps 1-6: return `assemblyResult{err: theError}`.

Steps 1-2 benefit from the Layer 2 LRU cache — on steady-state reconciles both vectors are cached, so the only network call is `GetArtifacts` (step 3), plus `CreateVector` (step 6) if drift is found.

---

## State machine

```
Reconcile(req):

  1. Synchronous setup (see above) → produces adapter, refs, config

  2. job, exists := r.getJob(nn)

     exists && job.generation != vt.Generation:
         job.Cancel() → r.removeJob(nn) → fall through to 3

     exists && job.generation == vt.Generation && not done:
         patch Assembling condition → return RequeueAfter(5s)

     exists && job.generation == vt.Generation && done:
         res := <-job.result → r.removeJob(nn) → apply result → patch status
         → return RequeueAfter(reconcileInterval)

  3. r.launchJob(nn, vt.Generation, assemblyFn)
     → patch Assembling condition → return RequeueAfter(5s)
```

After applying a terminal result, the reconcile returns `RequeueAfter(reconcileInterval)` (default 1 min) which drives the next periodic drift check. The 5s requeue is only for polling the inflight job.

---

## Correctness

| Mechanism | What it catches |
|-----------|-----------------|
| `job.generation != vt.Generation` at step 2 | Stale job from older spec; cancelled before new work launches |
| Generation re-check before status patch | Race where object was updated between poll and patch |
| `client.MergeFrom` optimistic lock | API server 409 on any conflicting write → backoff → requeue → self-healing |

---

## Status surface

Pure mapping functions translate results to Kubernetes types:

```go
func assemblyResultToCondition(res assemblyResult, vt *v1alpha1.VectorTemplate) metav1.Condition
func assemblyResultToEvent(res assemblyResult) (eventType, reason, action, message string)
func assemblingCondition(vt *v1alpha1.VectorTemplate) metav1.Condition
```

| Outcome | Condition reason | Message |
|---------|-----------------|---------|
| NoDrift | `NoDriftDetected` | `"No drift detected — vector version is still %s"` |
| Created | `VectorCreated` | `"Drift detected and new vector created — version %s"` |
| Failed  | `VectorCreationFailed` | `err.Error()` |
| In-progress | `Assembling` | `"Assembling vector in background"` |

`Assembling` is the only new observable state. Transient — replaced on the next tick.

---

## Tests

Existing drift specs (`Eventually` polls) tolerate async without changes.

New specs:

1. **Assembling → terminal**: trigger drift, assert `Assembling` within first requeue window, then assert `VectorCreated` after completion.

2. **Generation guard**: spec change while slow assembly is inflight (mock adapter with delay). Assert stale result discarded — status reflects the newer generation's outcome.

---

## File map

All changes in `internal/vectorassembly/internal/controller/`:

| File | Change |
|------|--------|
| `vectortemplate_controller.go` | Add types, reconciler fields, helper methods, rewrite `Reconcile` to state machine, add mapping functions |
| `vectortemplate_controller_test.go` | Add Assembling + generation guard specs |

No API type changes. No changes outside the controller package.
