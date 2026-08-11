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

// inflightJob tracks a single background assembly goroutine.
type inflightJob struct {
	generation int64
	cancel     context.CancelFunc
	result     chan assemblyResult // buffered cap 1, sent exactly once
}

func (j *inflightJob) done() bool { return len(j.result) > 0 }

// jobRegistry is a mutex-guarded map of inflight assembly jobs keyed by NamespacedName.
type jobRegistry struct {
	mu       sync.Mutex
	inflight map[types.NamespacedName]*inflightJob
}

func newJobRegistry() *jobRegistry {
	return &jobRegistry{inflight: make(map[types.NamespacedName]*inflightJob)}
}

func (r *jobRegistry) get(nn types.NamespacedName) (*inflightJob, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	job, ok := r.inflight[nn]
	return job, ok
}

// launch starts a background goroutine for nn/gen and registers it. Any previously
// registered job for nn must have been cancelled and removed by the caller first.
func (r *jobRegistry) launch(nn types.NamespacedName, gen int64, fn func(ctx context.Context) assemblyResult) {
	jobCtx, cancel := context.WithCancel(context.Background())
	job := &inflightJob{
		generation: gen,
		cancel:     cancel,
		result:     make(chan assemblyResult, 1),
	}

	r.mu.Lock()
	r.inflight[nn] = job
	r.mu.Unlock()

	go func() {
		defer cancel()
		job.result <- fn(jobCtx)
	}()
}

// remove removes the job from the registry and cancels its context. No-op if no
// job is registered for nn.
func (r *jobRegistry) remove(nn types.NamespacedName) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if job, ok := r.inflight[nn]; ok {
		job.cancel()
		delete(r.inflight, nn)
	}
}
