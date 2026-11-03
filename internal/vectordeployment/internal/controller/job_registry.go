package controller

import (
	"context"
	"slices"
	"sync"

	"k8s.io/apimachinery/pkg/types"
	"ocm.software/open-component-model/bindings/go/oci/compref"
)

type artifactManifestResult struct {
	artifacts []resolvedArtifactManifest
	err       error
}

type resolvedArtifactManifest struct {
	Reference compref.Ref
	Manifest  ArtifactManifest
}

type inflightJob struct {
	generation         int64
	uid                types.UID
	cancel             context.CancelFunc
	result             chan artifactManifestResult
	artifactReferences []string
}

func (j *inflightJob) done() bool {
	return len(j.result) > 0
}

func (j *inflightJob) matches(generation int64, uid types.UID, artifactReferences []string) bool {
	return j.generation == generation && j.uid == uid && slices.Equal(j.artifactReferences, artifactReferences)
}

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
	job, ok := r.inflight[nn]
	return job, ok
}

func (r *jobRegistry) launch(nn types.NamespacedName, generation int64, uid types.UID, artifactReferences []string,
	fn func(context.Context) artifactManifestResult) bool {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return false
	}

	jobCtx, cancel := context.WithCancel(r.parentCtx)
	job := &inflightJob{
		generation:         generation,
		uid:                uid,
		cancel:             cancel,
		result:             make(chan artifactManifestResult, 1),
		artifactReferences: append([]string(nil), artifactReferences...),
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

func (r *jobRegistry) remove(nn types.NamespacedName) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if job, ok := r.inflight[nn]; ok {
		job.cancel()
		delete(r.inflight, nn)
	}
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
	clear(r.inflight)
	r.mu.Unlock()
}
