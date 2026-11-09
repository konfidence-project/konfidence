package controller

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	ociv1 "ocm.software/open-component-model/bindings/go/oci/spec/repository/v1/oci"
)

type testVectorOcmPort struct {
	getVectorDescriptor func(context.Context, compref.Ref) (VectorDescriptor, error)
	getManifest         func(context.Context, compref.Ref) (ArtifactManifest, error)
}

func (p *testVectorOcmPort) GetVectorDescriptor(ctx context.Context, ref compref.Ref) (VectorDescriptor, error) {
	if p.getVectorDescriptor == nil {
		return VectorDescriptor{}, nil
	}
	return p.getVectorDescriptor(ctx, ref)
}

func (p *testVectorOcmPort) GetArtifactManifestByReference(ctx context.Context, ref compref.Ref) (ArtifactManifest, error) {
	return p.getManifest(ctx, ref)
}

func testArtifactRef(component, version string) compref.Ref {
	return compref.Ref{
		Repository: &ociv1.Repository{BaseUrl: "https://registry.example.com"},
		Component:  component,
		Version:    version,
	}
}

func testManifest(ref compref.Ref) ArtifactManifest {
	return ArtifactManifest{
		Type:       konfidence.MustArtifactType("test.konfidence.cloud"),
		AllowReuse: true,
		Resources: []OCMResource{{
			Name: ref.String(),
		}},
	}
}

func newManifestTestReconciler(port VectorOcmPort, perDeployment, global int) *VectorDeploymentReconciler {
	return &VectorDeploymentReconciler{
		OcmAdapter: port,
		jobs:       newJobRegistry(),
		MaxArtifactManifestDownloadsPerDeployment: perDeployment,
		manifestPollInterval:                      defaultArtifactManifestPollInterval,
		MaxGlobalArtifactManifestDownloads:        global,
		manifestDownloadTimeout:                   time.Second,
		manifestDownloadLimiter:                   newArtifactManifestDownloadLimiter(global),
	}
}

func updatePeak(peak *atomic.Int32, current int32) {
	for {
		old := peak.Load()
		if current <= old || peak.CompareAndSwap(old, current) {
			return
		}
	}
}

var _ = Describe("Artifact manifest jobs", func() {
	const (
		namespace = "default"
		name      = "vector"
	)

	var nn types.NamespacedName

	BeforeEach(func() {
		nn = types.NamespacedName{Namespace: namespace, Name: name}
	})

	It("reuses a job when references only differ in order", func() {
		refA := testArtifactRef("example.com/a", "1.0.0")
		refB := testArtifactRef("example.com/b", "2.0.0")
		input := []compref.Ref{refB, refA}

		release := make(chan struct{})
		started := make(chan compref.Ref, 2)
		var calls atomic.Int32

		port := &testVectorOcmPort{
			getManifest: func(ctx context.Context, ref compref.Ref) (ArtifactManifest, error) {
				calls.Add(1)
				started <- ref

				select {
				case <-release:
					return testManifest(ref), nil
				case <-ctx.Done():
					return ArtifactManifest{}, ctx.Err()
				}
			},
		}

		r := newManifestTestReconciler(port, 2, 2)
		_, retryAfter := r.resolveArtifactManifestsAsync(nn, 1, types.UID("uid-1"), input)
		Expect(retryAfter).To(Equal(defaultArtifactManifestPollInterval))

		Eventually(started).Should(Receive())
		Eventually(started).Should(Receive())

		_, retryAfter = r.resolveArtifactManifestsAsync(nn, 1, types.UID("uid-1"), []compref.Ref{refA, refB})
		Expect(retryAfter).To(Equal(defaultArtifactManifestPollInterval))
		Expect(calls.Load()).To(Equal(int32(2)))

		By("not mutating the caller's reference slice")
		Expect(input[0].String()).To(Equal(refB.String()))
		Expect(input[1].String()).To(Equal(refA.String()))

		close(release)

		var result artifactManifestResult
		Eventually(func() time.Duration {
			result, retryAfter = r.resolveArtifactManifestsAsync(nn, 1, types.UID("uid-1"), []compref.Ref{refA, refB})
			return retryAfter
		}).Should(BeZero())

		Expect(result.err).NotTo(HaveOccurred())
		Expect(result.artifacts).To(HaveLen(2))

		Expect(result.artifacts[0].Reference.String()).To(Equal(refA.String()))
		Expect(result.artifacts[0].Manifest.Resources[0].Name).To(Equal(refA.String()))
		Expect(result.artifacts[1].Reference.String()).To(Equal(refB.String()))
		Expect(result.artifacts[1].Manifest.Resources[0].Name).To(Equal(refB.String()))

		By("reusing the successful result without another download")
		cachedResult, retryAfter := r.resolveArtifactManifestsAsync(nn, 1, types.UID("uid-1"), []compref.Ref{refB, refA})
		Expect(retryAfter).To(BeZero())
		Expect(cachedResult.err).NotTo(HaveOccurred())
		Expect(cachedResult.artifacts).To(HaveLen(2))
		Expect(calls.Load()).To(Equal(int32(2)))
	})

	DescribeTable("replaces stale jobs",
		func(oldGeneration, newGeneration int64, oldUID, newUID types.UID, oldRefs, newRefs []compref.Ref) {
			firstStarted := make(chan struct{})
			firstCanceled := make(chan struct{})
			var calls atomic.Int32

			port := &testVectorOcmPort{
				getManifest: func(ctx context.Context, ref compref.Ref) (ArtifactManifest, error) {
					if calls.Add(1) == 1 {
						close(firstStarted)
						<-ctx.Done()
						close(firstCanceled)
						return ArtifactManifest{}, ctx.Err()
					}
					return testManifest(ref), nil
				},
			}

			r := newManifestTestReconciler(port, 1, 1)

			_, retryAfter := r.resolveArtifactManifestsAsync(nn, oldGeneration, oldUID, oldRefs)
			Expect(retryAfter).To(Equal(defaultArtifactManifestPollInterval))
			Eventually(firstStarted).Should(BeClosed())

			_, retryAfter = r.resolveArtifactManifestsAsync(nn, newGeneration, newUID, newRefs)
			Expect(retryAfter).To(Equal(defaultArtifactManifestPollInterval))
			Eventually(firstCanceled).Should(BeClosed())

			var result artifactManifestResult
			Eventually(func() time.Duration {
				result, retryAfter = r.resolveArtifactManifestsAsync(nn, newGeneration, newUID, newRefs)
				return retryAfter
			}).Should(BeZero())

			Expect(result.err).NotTo(HaveOccurred())
			Expect(result.artifacts).To(HaveLen(1))
			Expect(result.artifacts[0].Reference.String()).To(Equal(newRefs[0].String()))
		},

		Entry(
			"when generation changes",
			int64(1), int64(2),
			types.UID("uid"), types.UID("uid"),
			[]compref.Ref{testArtifactRef("example.com/a", "1.0.0")},
			[]compref.Ref{testArtifactRef("example.com/a", "1.0.0")},
		),

		Entry(
			"when UID changes",
			int64(1), int64(1),
			types.UID("old"), types.UID("new"),
			[]compref.Ref{testArtifactRef("example.com/a", "1.0.0")},
			[]compref.Ref{testArtifactRef("example.com/a", "1.0.0")},
		),

		Entry(
			"when references change",
			int64(1), int64(1),
			types.UID("uid"), types.UID("uid"),
			[]compref.Ref{testArtifactRef("example.com/a", "1.0.0")},
			[]compref.Ref{testArtifactRef("example.com/a", "2.0.0")},
		),
	)

	It("cancels an obsolete job when references become empty", func() {
		started := make(chan struct{})
		canceled := make(chan struct{})

		port := &testVectorOcmPort{
			getManifest: func(ctx context.Context, ref compref.Ref) (ArtifactManifest, error) {
				close(started)
				<-ctx.Done()
				close(canceled)
				return ArtifactManifest{}, ctx.Err()
			},
		}

		r := newManifestTestReconciler(port, 1, 1)
		_, retryAfter := r.resolveArtifactManifestsAsync(nn, 1, types.UID("uid"), []compref.Ref{testArtifactRef("example.com/a", "1.0.0")})
		Expect(retryAfter).To(Equal(defaultArtifactManifestPollInterval))
		Eventually(started).Should(BeClosed())

		result, retryAfter := r.resolveArtifactManifestsAsync(nn, 1, types.UID("uid"), nil)
		Expect(retryAfter).To(BeZero())
		Expect(result.err).NotTo(HaveOccurred())
		Eventually(canceled).Should(BeClosed())

		_, exists := r.jobs.get(nn)
		Expect(exists).To(BeFalse())
	})

	It("backs off repeated failed jobs before launching another download", func() {
		ref := testArtifactRef("example.com/a", "1.0.0")
		references := []compref.Ref{ref}
		var calls atomic.Int32

		port := &testVectorOcmPort{
			getManifest: func(context.Context, compref.Ref) (ArtifactManifest, error) {
				calls.Add(1)
				return ArtifactManifest{}, errors.New("registry unavailable")
			},
		}

		r := newManifestTestReconciler(port, 1, 1)
		now := time.Date(2026, time.November, 4, 12, 0, 0, 0, time.UTC)
		r.jobs.now = func() time.Time {
			return now
		}

		_, retryAfter := r.resolveArtifactManifestsAsync(nn, 1, types.UID("uid"), references)
		Expect(retryAfter).To(Equal(defaultArtifactManifestPollInterval))
		Eventually(calls.Load).Should(Equal(int32(1)))
		var result artifactManifestResult
		Eventually(func() time.Duration {
			result, retryAfter = r.resolveArtifactManifestsAsync(nn, 1, types.UID("uid"), references)
			return retryAfter
		}).Should(BeZero())

		Expect(result.err).To(MatchError(ContainSubstring("registry unavailable")))
		Expect(retryAfter).To(BeZero())

		_, retryAfter = r.resolveArtifactManifestsAsync(nn, 1, types.UID("uid"), references)
		Expect(retryAfter).To(Equal(defaultArtifactManifestRetryBaseDelay))
		Expect(calls.Load()).To(Equal(int32(1)))

		now = now.Add(defaultArtifactManifestRetryBaseDelay - time.Millisecond)
		_, retryAfter = r.resolveArtifactManifestsAsync(nn, 1, types.UID("uid"), references)
		Expect(retryAfter).To(Equal(time.Millisecond))
		Expect(calls.Load()).To(Equal(int32(1)))

		now = now.Add(time.Millisecond)
		_, retryAfter = r.resolveArtifactManifestsAsync(nn, 1, types.UID("uid"), references)
		Expect(retryAfter).To(Equal(defaultArtifactManifestPollInterval))
		Eventually(calls.Load).Should(Equal(int32(2)))
		Eventually(func() time.Duration {
			result, retryAfter = r.resolveArtifactManifestsAsync(nn, 1, types.UID("uid"), references)
			return retryAfter
		}).Should(BeZero())

		Expect(result.err).To(HaveOccurred())
		Expect(retryAfter).To(BeZero())

		_, retryAfter = r.resolveArtifactManifestsAsync(nn, 1, types.UID("uid"), references)
		Expect(retryAfter).To(Equal(2 * defaultArtifactManifestRetryBaseDelay))
		Expect(calls.Load()).To(Equal(int32(2)))
	})

	It("resets manifest retry backoff when the job identity changes", func() {
		ref := testArtifactRef("example.com/a", "1.0.0")
		references := []compref.Ref{ref}
		var calls atomic.Int32

		port := &testVectorOcmPort{
			getManifest: func(context.Context, compref.Ref) (ArtifactManifest, error) {
				calls.Add(1)
				return ArtifactManifest{}, errors.New("registry unavailable")
			},
		}

		r := newManifestTestReconciler(port, 1, 1)

		_, retryAfter := r.resolveArtifactManifestsAsync(nn, 1, types.UID("uid"), references)
		Expect(retryAfter).To(Equal(defaultArtifactManifestPollInterval))
		var result artifactManifestResult
		Eventually(func() time.Duration {
			result, retryAfter = r.resolveArtifactManifestsAsync(nn, 1, types.UID("uid"), references)
			return retryAfter
		}).Should(BeZero())

		Expect(result.err).To(HaveOccurred())
		Expect(retryAfter).To(BeZero())

		_, retryAfter = r.resolveArtifactManifestsAsync(nn, 2, types.UID("uid"), references)
		Expect(retryAfter).To(Equal(defaultArtifactManifestPollInterval))
		Eventually(calls.Load).Should(Equal(int32(2)))
	})

	It("limits parallel downloads per VectorDeployment", func() {
		refs := []compref.Ref{
			testArtifactRef("example.com/a", "1.0.0"),
			testArtifactRef("example.com/b", "1.0.0"),
			testArtifactRef("example.com/c", "1.0.0"),
			testArtifactRef("example.com/d", "1.0.0"),
		}

		release := make(chan struct{})
		var active atomic.Int32
		var peak atomic.Int32
		var calls atomic.Int32

		port := &testVectorOcmPort{
			getManifest: func(ctx context.Context, ref compref.Ref) (ArtifactManifest, error) {
				calls.Add(1)
				current := active.Add(1)
				updatePeak(&peak, current)
				defer active.Add(-1)

				select {
				case <-release:
					return testManifest(ref), nil
				case <-ctx.Done():
					return ArtifactManifest{}, ctx.Err()
				}
			},
		}

		r := newManifestTestReconciler(port, 2, 8)
		resultChannel := make(chan artifactManifestResult, 1)

		go func() {
			resultChannel <- r.downloadArtifactManifests(context.Background(), refs)
		}()

		Eventually(calls.Load).Should(Equal(int32(2)))
		Consistently(peak.Load, 100*time.Millisecond).Should(BeNumerically("<=", 2))
		close(release)

		var result artifactManifestResult
		Eventually(resultChannel).Should(Receive(&result))
		Expect(result.err).NotTo(HaveOccurred())
		Expect(result.artifacts).To(HaveLen(4))
		Expect(peak.Load()).To(Equal(int32(2)))
	})

	It("enforces the global limit across deployments", func() {
		release := make(chan struct{})
		var active atomic.Int32
		var peak atomic.Int32
		var calls atomic.Int32

		port := &testVectorOcmPort{
			getManifest: func(ctx context.Context, ref compref.Ref) (ArtifactManifest, error) {
				calls.Add(1)
				current := active.Add(1)
				updatePeak(&peak, current)
				defer active.Add(-1)

				select {
				case <-release:
					return testManifest(ref), nil
				case <-ctx.Done():
					return ArtifactManifest{}, ctx.Err()
				}
			},
		}

		r := newManifestTestReconciler(port, 4, 2)
		results := make(chan artifactManifestResult, 2)

		go func() {
			results <- r.downloadArtifactManifests(context.Background(), []compref.Ref{
				testArtifactRef("example.com/a", "1.0.0"),
				testArtifactRef("example.com/b", "1.0.0"),
			})
		}()
		go func() {
			results <- r.downloadArtifactManifests(context.Background(), []compref.Ref{
				testArtifactRef("example.com/c", "1.0.0"),
				testArtifactRef("example.com/d", "1.0.0"),
			})
		}()

		Eventually(calls.Load).Should(Equal(int32(2)))
		Consistently(peak.Load, 100*time.Millisecond).Should(BeNumerically("<=", 2))

		close(release)

		var first, second artifactManifestResult
		Eventually(results).Should(Receive(&first))
		Eventually(results).Should(Receive(&second))
		Expect(first.err).NotTo(HaveOccurred())
		Expect(second.err).NotTo(HaveOccurred())
		Expect(peak.Load()).To(Equal(int32(2)))
	})

	It("starts the operation timeout after acquiring the global slot", func() {
		ref := testArtifactRef("example.com/a", "1.0.0")
		var calls atomic.Int32

		port := &testVectorOcmPort{
			getManifest: func(ctx context.Context, ref compref.Ref) (ArtifactManifest, error) {
				calls.Add(1)
				<-ctx.Done()
				return ArtifactManifest{}, ctx.Err()
			},
		}

		r := newManifestTestReconciler(port, 1, 1)
		r.manifestDownloadTimeout = 50 * time.Millisecond

		release, err := r.manifestDownloadLimiter.acquire(context.Background())
		Expect(err).NotTo(HaveOccurred())

		resultChannel := make(chan artifactManifestResult, 1)
		go func() {
			resultChannel <- r.downloadArtifactManifests(context.Background(), []compref.Ref{ref})
		}()

		Consistently(calls.Load, 100*time.Millisecond).Should(Equal(int32(0)))
		release()

		Eventually(calls.Load).Should(Equal(int32(1)))

		var result artifactManifestResult
		Eventually(resultChannel).Should(Receive(&result))
		Expect(errors.Is(result.err, context.DeadlineExceeded)).To(BeTrue())
		Expect(result.err.Error()).To(ContainSubstring(ref.String()))
	})

	It("preserves PermanentError through download wrapping", func() {
		ref := testArtifactRef("example.com/a", "1.0.0")
		permanent := &PermanentError{
			Reason: konfidence.VectorDeploymentStalledReasonManifestMissing,
			Err:    errors.New("manifest missing"),
		}

		port := &testVectorOcmPort{
			getManifest: func(context.Context, compref.Ref) (ArtifactManifest, error) {
				return ArtifactManifest{}, permanent
			},
		}

		r := newManifestTestReconciler(port, 1, 1)
		result := r.downloadArtifactManifests(context.Background(), []compref.Ref{ref})

		var actual *PermanentError
		Expect(errors.As(result.err, &actual)).To(BeTrue())
		Expect(actual.Reason).To(Equal(permanent.Reason))
	})

	It("prefers a PermanentError over a sibling cancellation", func() {
		canceledRef := testArtifactRef("example.com/canceled", "1.0.0")
		permanentRef := testArtifactRef("example.com/permanent", "1.0.0")
		permanentStarted := make(chan struct{})

		permanent := &PermanentError{
			Reason: konfidence.VectorDeploymentStalledReasonManifestMissing,
			Err:    errors.New("manifest missing"),
		}

		port := &testVectorOcmPort{
			getManifest: func(ctx context.Context, ref compref.Ref) (ArtifactManifest, error) {
				if ref.String() == permanentRef.String() {
					close(permanentStarted)
					<-ctx.Done()
					return ArtifactManifest{}, permanent
				}

				<-permanentStarted
				return ArtifactManifest{}, context.Canceled
			},
		}

		r := newManifestTestReconciler(port, 2, 2)
		result := r.downloadArtifactManifests(
			context.Background(),
			[]compref.Ref{canceledRef, permanentRef},
		)

		var actual *PermanentError
		Expect(errors.As(result.err, &actual)).To(BeTrue())
		Expect(actual).To(BeIdenticalTo(permanent))
		Expect(result.err.Error()).To(ContainSubstring(permanentRef.String()))
	})

	It("converts an artifact manifest download panic into an error", func() {
		ref := testArtifactRef("example.com/a", "1.0.0")
		var calls atomic.Int32

		port := &testVectorOcmPort{
			getManifest: func(context.Context, compref.Ref) (ArtifactManifest, error) {
				if calls.Add(1) == 1 {
					panic("malformed registry content")
				}
				return testManifest(ref), nil
			},
		}

		r := newManifestTestReconciler(port, 1, 1)

		result := r.downloadArtifactManifests(context.Background(), []compref.Ref{ref})
		Expect(result.err).To(HaveOccurred())
		Expect(result.err.Error()).To(ContainSubstring("malformed registry content"))
		Expect(result.err.Error()).To(ContainSubstring(ref.String()))

		By("releasing the download slot after the panic")
		result = r.downloadArtifactManifests(context.Background(), []compref.Ref{ref})
		Expect(result.err).NotTo(HaveOccurred())
		Expect(result.artifacts).To(HaveLen(1))
	})

	DescribeTable("invalidates a successful cached result when its identity changes",
		func(oldGeneration, newGeneration int64, oldUID, newUID types.UID, oldRefs, newRefs []compref.Ref) {
			var calls atomic.Int32

			port := &testVectorOcmPort{
				getManifest: func(_ context.Context, ref compref.Ref) (ArtifactManifest, error) {
					calls.Add(1)
					return testManifest(ref), nil
				},
			}

			r := newManifestTestReconciler(port, 1, 1)

			_, retryAfter := r.resolveArtifactManifestsAsync(nn, oldGeneration, oldUID, oldRefs)
			Expect(retryAfter).To(Equal(defaultArtifactManifestPollInterval))

			var result artifactManifestResult
			Eventually(func() time.Duration {
				result, retryAfter = r.resolveArtifactManifestsAsync(nn, oldGeneration, oldUID, oldRefs)
				return retryAfter
			}).Should(BeZero())

			Expect(result.err).NotTo(HaveOccurred())
			Expect(retryAfter).To(BeZero())
			Expect(calls.Load()).To(Equal(int32(1)))

			_, retryAfter = r.resolveArtifactManifestsAsync(nn, newGeneration, newUID, newRefs)
			Expect(retryAfter).To(Equal(defaultArtifactManifestPollInterval))
			Eventually(calls.Load).Should(Equal(int32(2)))
		},
		Entry(
			"when generation changes",
			int64(1), int64(2),
			types.UID("uid"), types.UID("uid"),
			[]compref.Ref{testArtifactRef("example.com/a", "1.0.0")},
			[]compref.Ref{testArtifactRef("example.com/a", "1.0.0")},
		),
		Entry(
			"when UID changes",
			int64(1), int64(1),
			types.UID("old"), types.UID("new"),
			[]compref.Ref{testArtifactRef("example.com/a", "1.0.0")},
			[]compref.Ref{testArtifactRef("example.com/a", "1.0.0")},
		),
		Entry(
			"when references change",
			int64(1), int64(1),
			types.UID("uid"), types.UID("uid"),
			[]compref.Ref{testArtifactRef("example.com/a", "1.0.0")},
			[]compref.Ref{testArtifactRef("example.com/a", "2.0.0")},
		),
	)
})

var _ = Describe("Ready generation matching", func() {
	DescribeTable("reports readiness for the current generation",
		func(
			condition *metav1.Condition,
			generation int64,
			expected bool,
		) {
			vd := &konfidence.VectorDeployment{
				ObjectMeta: metav1.ObjectMeta{Generation: generation},
			}
			if condition != nil {
				vd.Status.Conditions = []metav1.Condition{*condition}
			}

			Expect(isReadyForCurrentGeneration(vd)).To(Equal(expected))
		},

		Entry("without a condition", nil, int64(2), false),

		Entry(
			"when Ready is false",
			&metav1.Condition{
				Type:               konfidence.VectorReadyCondition,
				Status:             metav1.ConditionFalse,
				ObservedGeneration: 2,
			},
			int64(2),
			false,
		),

		Entry(
			"when Ready is true for the current generation",
			&metav1.Condition{
				Type:               konfidence.VectorReadyCondition,
				Status:             metav1.ConditionTrue,
				ObservedGeneration: 2,
			},
			int64(2),
			true,
		),

		Entry(
			"when Ready is true for an old generation",
			&metav1.Condition{
				Type:               konfidence.VectorReadyCondition,
				Status:             metav1.ConditionTrue,
				ObservedGeneration: 1,
			},
			int64(2),
			false,
		),
	)
})
