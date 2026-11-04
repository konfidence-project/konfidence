package controller

import (
	"context"
	"slices"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"ocm.software/open-component-model/bindings/go/oci/compref"
)

const (
	defaultArtifactManifestRetryBaseDelay = 5 * time.Millisecond
	defaultArtifactManifestRetryMaxDelay  = 1000 * time.Second
)

type artifactManifestResult struct {
	artifacts []resolvedArtifactManifest
	err       error
}

type resolvedArtifactManifest struct {
	Reference compref.Ref
	Manifest  ArtifactManifest
}

type manifestJobIdentity struct {
	generation         int64
	uid                types.UID
	artifactReferences []string
}

func newManifestJobIdentity(generation int64, uid types.UID, artifactReferences []string) manifestJobIdentity {
	return manifestJobIdentity{
		generation:         generation,
		uid:                uid,
		artifactReferences: append([]string(nil), artifactReferences...),
	}
}

func (i manifestJobIdentity) matches(generation int64, uid types.UID, artifactReferences []string) bool {
	return i.generation == generation && i.uid == uid && slices.Equal(i.artifactReferences, artifactReferences)
}

type inflightJob struct {
	manifestJobIdentity
	cancel context.CancelFunc
	result chan artifactManifestResult
}

func (j *inflightJob) done() bool {
	return len(j.result) > 0
}

type manifestRetry struct {
	manifestJobIdentity
	notBefore time.Time
}

type jobRegistry struct {
	mu           sync.Mutex
	wg           sync.WaitGroup
	inflight     map[types.NamespacedName]*inflightJob
	parentCtx    context.Context
	retries      map[types.NamespacedName]*manifestRetry
	retryLimiter workqueue.TypedRateLimiter[types.NamespacedName]
	now          func() time.Time
	cancelParent context.CancelFunc
	closed       bool
}

func newJobRegistry() *jobRegistry {
	parentCtx, cancelParent := context.WithCancel(context.Background())
	return &jobRegistry{
		inflight: make(map[types.NamespacedName]*inflightJob),
		retries:  make(map[types.NamespacedName]*manifestRetry),
		retryLimiter: workqueue.NewTypedItemExponentialFailureRateLimiter[types.NamespacedName](
			defaultArtifactManifestRetryBaseDelay,
			defaultArtifactManifestRetryMaxDelay,
		),
		now:          time.Now,
		parentCtx:    parentCtx,
		cancelParent: cancelParent,
	}
}

func (r *jobRegistry) get(nn types.NamespacedName) (*inflightJob, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	job, ok := r.inflight[nn]
	return job, ok
}

func (r *jobRegistry) launch(nn types.NamespacedName, generation int64, uid types.UID, artifactReferences []string,
	fn func(context.Context) artifactManifestResult) (bool, time.Duration) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return false, 0
	}

	if retry, exists := r.retries[nn]; exists {
		if !retry.matches(generation, uid, artifactReferences) {
			r.clearRetryLocked(nn)
		} else if retryAfter := retry.notBefore.Sub(r.now()); retryAfter > 0 {
			r.mu.Unlock()
			return false, retryAfter
		}
	}

	jobCtx, cancel := context.WithCancel(r.parentCtx)
	job := &inflightJob{
		manifestJobIdentity: newManifestJobIdentity(generation, uid, artifactReferences),
		cancel:              cancel,
		result:              make(chan artifactManifestResult, 1),
	}

	r.inflight[nn] = job
	r.wg.Add(1)
	r.mu.Unlock()

	go func() {
		defer r.wg.Done()
		defer cancel()
		job.result <- fn(jobCtx)
	}()

	return true, 0
}

// complete removes a finished job and updates its retry state. Failed jobs are
// delayed independently of controller-runtime's reconcile queue because a
// RequeueAfter result causes controller-runtime to forget its own failure count.
func (r *jobRegistry) complete(nn types.NamespacedName, job *inflightJob, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	current, exists := r.inflight[nn]
	if !exists || current != job {
		return
	}

	job.cancel()
	delete(r.inflight, nn)

	if err == nil {
		r.clearRetryLocked(nn)
		return
	}

	delay := r.retryLimiter.When(nn)
	r.retries[nn] = &manifestRetry{
		manifestJobIdentity: job.manifestJobIdentity,
		notBefore:           r.now().Add(delay),
	}
}

func (r *jobRegistry) remove(nn types.NamespacedName) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if job, ok := r.inflight[nn]; ok {
		job.cancel()
		delete(r.inflight, nn)
	}

	r.clearRetryLocked(nn)
}

func (r *jobRegistry) clearRetryLocked(nn types.NamespacedName) {
	delete(r.retries, nn)
	r.retryLimiter.Forget(nn)
}

// shutdown prevents active jobs from outliving the controller manager.
func (r *jobRegistry) shutdown() {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		r.wg.Wait()
		return
	}

	// setting closed while holding the mutex prevents launch from adding to
	// the WaitGroup after shutdown starts waiting
	r.closed = true
	r.cancelParent()
	r.mu.Unlock()
	r.wg.Wait()

	r.mu.Lock()
	for nn := range r.retries {
		r.retryLimiter.Forget(nn)
	}

	clear(r.inflight)
	clear(r.retries)
	r.mu.Unlock()
}
