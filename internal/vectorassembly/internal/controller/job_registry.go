package controller

import (
	"context"
	"sync"

	"k8s.io/apimachinery/pkg/types"
)

// assemblyResult is the outcome of a background assembly job.
type assemblyResult struct {
	latestVector  string // set when drift was resolved and a new vector was written
	vectorVersion string // current (no-drift) or new (created) version for condition messages
	componentName string // for log/event messages
	err           error
}

func (r assemblyResult) failed() bool  { return r.err != nil }
func (r assemblyResult) created() bool { return r.err == nil && r.latestVector != "" }

type assemblyJobIdentity struct {
	generation    int64
	uid           types.UID
	baseVectorRef string
}

func (i assemblyJobIdentity) matchesObject(generation int64, uid types.UID) bool {
	return i.generation == generation && i.uid == uid
}

func (i assemblyJobIdentity) matches(generation int64, uid types.UID, baseVectorRef string) bool {
	return i.matchesObject(generation, uid) && i.baseVectorRef == baseVectorRef
}

// inflightJob tracks a single background assembly goroutine.
type inflightJob struct {
	assemblyJobIdentity
	cancel context.CancelFunc
	result chan assemblyResult // buffered cap 1, sent exactly once
}

// jobRegistry is a mutex-guarded map of inflight assembly jobs keyed by NamespacedName.
type jobRegistry struct {
	mu           sync.Mutex
	wg           sync.WaitGroup
	inflight     map[types.NamespacedName]*inflightJob
	parentCtx    context.Context
	cancelParent context.CancelFunc
	closed       bool
}

func newJobRegistry() *jobRegistry {
	parentCtx, cancelParent := context.WithCancel(context.Background())

	return &jobRegistry{
		inflight:     make(map[types.NamespacedName]*inflightJob),
		parentCtx:    parentCtx,
		cancelParent: cancelParent,
	}
}

func (r *jobRegistry) get(nn types.NamespacedName) (*inflightJob, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	job, exists := r.inflight[nn]
	return job, exists
}

// launch starts and registers a background job for the object identity.
// Any existing job for the same key is canceled before it is replaced.
func (r *jobRegistry) launch(nn types.NamespacedName, generation int64, uid types.UID, baseVectorRef string,
	fn func(context.Context) assemblyResult) bool {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return false
	}

	jobCtx, cancel := context.WithCancel(r.parentCtx)
	job := &inflightJob{
		generation:    generation,
		uid:           uid,
		baseVectorRef: baseVectorRef,
		cancel:        cancel,
		result:        make(chan assemblyResult, 1),
	}

	// Defensive replacement: callers normally remove stale jobs first.
	if previous, exists := r.inflight[nn]; exists && previous.cancel != nil {
		previous.cancel()
	}

	r.inflight[nn] = job
	r.wg.Add(1)
	r.mu.Unlock()

	go func() {
		defer r.wg.Done()
		defer cancel()

		job.result <- fn(jobCtx)
	}()

	return true
}

// completedResult atomically checks and consumes a completed result.
// Unlike VectorDeployment results, assembly results are not cached because
// the next scheduled reconciliation must perform drift detection again.
func (r *jobRegistry) completedResult(nn types.NamespacedName, job *inflightJob) (assemblyResult, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	current, exists := r.inflight[nn]
	if !exists || current != job {
		return assemblyResult{}, false
	}

	select {
	case result := <-job.result:
		if job.cancel != nil {
			job.cancel()
			job.cancel = nil
		}
		delete(r.inflight, nn)
		return result, true
	default:
		return assemblyResult{}, false
	}
}

// remove removes the job from the registry and cancels its context. No-op if no
// job is registered for nn.
func (r *jobRegistry) remove(nn types.NamespacedName) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if job, exists := r.inflight[nn]; exists {
		if job.cancel != nil {
			job.cancel()
		}
		delete(r.inflight, nn)
	}
}

func (r *jobRegistry) shutdown() {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		r.wg.Wait()
		return
	}

	// Holding the mutex prevents launch from adding to the WaitGroup after
	// shutdown starts waiting.
	r.closed = true
	r.cancelParent()
	r.mu.Unlock()

	r.wg.Wait()

	r.mu.Lock()
	clear(r.inflight)
	r.mu.Unlock()
}
