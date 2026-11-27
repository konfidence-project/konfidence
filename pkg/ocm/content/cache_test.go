package content

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	testCacheSize = 100
	testTagTTL    = 5 * time.Minute

	refCommonPrefix = "registry.example.com//github.com/example/foo"
	semverRef       = refCommonPrefix + ":1.0.0"
	aliasRef        = refCommonPrefix + ":latest"
)

var digestRef = refCommonPrefix + ":latest@sha256:" + strings.Repeat("a", 64)

var (
	testVectorContent   = VectorContent{Components: []VectorComponent{{ComponentName: "a", Version: "1.0.0"}}}
	testArtifactContent = ArtifactContent{Type: "some.type", AllowReuse: true}
)

type mockOcmReader struct {
	vectorCalls   atomic.Int64
	artifactCalls atomic.Int64

	err  error
	gate chan struct{}
}

func (r *mockOcmReader) GetVectorContent(_ context.Context, _ string) (VectorContent, error) {
	r.vectorCalls.Add(1)
	if r.gate != nil {
		<-r.gate
	}
	if r.err != nil {
		return VectorContent{}, r.err
	}
	return testVectorContent, nil
}

func (r *mockOcmReader) GetArtifactContent(_ context.Context, _ string) (ArtifactContent, error) {
	r.artifactCalls.Add(1)
	if r.gate != nil {
		<-r.gate
	}
	if r.err != nil {
		return ArtifactContent{}, r.err
	}
	return testArtifactContent, nil
}

type contentType int

const (
	vectorContent contentType = iota
	artifactContent
)

func (c contentType) String() string {
	if c == vectorContent {
		return "vectors"
	}
	return "artifacts"
}

func get(ctx context.Context, c contentType, cache *cachingReader, ref string) error {
	if c == vectorContent {
		_, err := cache.GetVectorContent(ctx, ref)
		return err
	}
	_, err := cache.GetArtifactContent(ctx, ref)
	return err
}

func callCount(c contentType, r *mockOcmReader) int64 {
	if c == vectorContent {
		return r.vectorCalls.Load()
	}
	return r.artifactCalls.Load()
}

var _ = Describe("Caching OCM content reader", func() {
	var (
		ctx        = context.Background()
		mockReader *mockOcmReader
	)

	BeforeEach(func() {
		mockReader = &mockOcmReader{}
	})

	newCache := func() *cachingReader {
		r, err := NewCachingReader(mockReader, testCacheSize, testTagTTL)
		Expect(err).ToNot(HaveOccurred())
		return r.(*cachingReader)
	}

	newCacheAt := func(now *time.Time) *cachingReader {
		c := newCache()
		c.clock = func() time.Time {
			return *now
		}
		return c
	}

	It("calls inner reader on the first vector request and returns its content", func() {
		cache := newCache()
		vc, err := cache.GetVectorContent(ctx, semverRef)
		Expect(err).ToNot(HaveOccurred())
		Expect(vc.Components).To(Equal(testVectorContent.Components))
		Expect(mockReader.vectorCalls.Load()).To(Equal(int64(1)))
	})

	It("does not call inner reader on vector cache hit and returns the content from the cache", func() {
		cache := newCache()
		_, err := cache.GetVectorContent(ctx, semverRef)
		Expect(err).ToNot(HaveOccurred())
		vc, err := cache.GetVectorContent(ctx, semverRef)
		Expect(err).ToNot(HaveOccurred())
		Expect(vc.Components).To(Equal(testVectorContent.Components))
		Expect(mockReader.vectorCalls.Load()).To(Equal(int64(1)))
	})

	It("calls inner reader again on second request for vector with a different ref", func() {
		cache := newCache()
		_, _ = cache.GetVectorContent(ctx, semverRef)
		_, _ = cache.GetVectorContent(ctx, aliasRef)
		Expect(mockReader.vectorCalls.Load()).To(Equal(int64(2)))
	})

	It("calls inner reader on the first artifact request and returns its content", func() {
		cache := newCache()
		ac, err := cache.GetArtifactContent(ctx, semverRef)
		Expect(err).ToNot(HaveOccurred())
		Expect(ac).To(Equal(testArtifactContent))
		Expect(mockReader.artifactCalls.Load()).To(Equal(int64(1)))
	})

	It("does not call inner reader on artifact cache hit and returns the content from the cache", func() {
		cache := newCache()
		_, err := cache.GetArtifactContent(ctx, semverRef)
		Expect(err).ToNot(HaveOccurred())
		ac, err := cache.GetArtifactContent(ctx, semverRef)
		Expect(err).ToNot(HaveOccurred())
		Expect(ac).To(Equal(testArtifactContent))
		Expect(mockReader.artifactCalls.Load()).To(Equal(int64(1)))
	})

	It("calls inner reader again on second request for artifact with a different ref", func() {
		cache := newCache()
		_, _ = cache.GetArtifactContent(ctx, semverRef)
		_, _ = cache.GetArtifactContent(ctx, aliasRef)
		Expect(mockReader.artifactCalls.Load()).To(Equal(int64(2)))
	})

	It("caches vector and artifact content separately for matching refs", func() {
		cache := newCache()
		vc, _ := cache.GetVectorContent(ctx, semverRef)
		ac, _ := cache.GetArtifactContent(ctx, semverRef)
		Expect(vc.Components).To(Equal(testVectorContent.Components))
		Expect(ac).To(Equal(testArtifactContent))
		Expect(mockReader.vectorCalls.Load()).To(Equal(int64(1)))
		Expect(mockReader.artifactCalls.Load()).To(Equal(int64(1)))

		_, _ = cache.GetArtifactContent(ctx, semverRef)
		_, _ = cache.GetArtifactContent(ctx, semverRef)
		Expect(mockReader.artifactCalls.Load()).To(Equal(int64(1)))
		Expect(mockReader.vectorCalls.Load()).To(Equal(int64(1)))
	})

	It("collapses concurrent calls for the same vector ref into one inner reader call when not already in cache", func() {
		mockReader.gate = make(chan struct{})
		cache := newCache()

		const n = 5
		var wg sync.WaitGroup
		wg.Add(n)
		for i := 0; i < n; i++ {
			go func() {
				defer wg.Done()
				vc, err := cache.GetVectorContent(ctx, semverRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(vc.Components).To(Equal(testVectorContent.Components))
			}()
		}

		Eventually(mockReader.vectorCalls.Load).Should(Equal(int64(1)))
		Consistently(mockReader.vectorCalls.Load).Should(Equal(int64(1)))
		close(mockReader.gate)
		wg.Wait()

		Expect(mockReader.vectorCalls.Load()).To(Equal(int64(1)))
	})

	It("collapses concurrent calls for the same artifact ref into one inner reader call when not already in cache", func() {
		mockReader.gate = make(chan struct{})
		cache := newCache()

		const n = 5
		var wg sync.WaitGroup
		wg.Add(n)
		for i := 0; i < n; i++ {
			go func() {
				defer wg.Done()
				ac, err := cache.GetArtifactContent(ctx, semverRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ac).To(Equal(testArtifactContent))
			}()
		}

		Eventually(mockReader.artifactCalls.Load).Should(Equal(int64(1)))
		Consistently(mockReader.artifactCalls.Load).Should(Equal(int64(1)))
		close(mockReader.gate)
		wg.Wait()

		Expect(mockReader.artifactCalls.Load()).To(Equal(int64(1)))
	})

	It("lets a cancelled vector caller return without blocking the others sharing its fetch", func() {
		mockReader.gate = make(chan struct{})
		cache := newCache()

		var runnerWg sync.WaitGroup
		runnerWg.Add(1)
		go func() {
			defer runnerWg.Done()
			vc, err := cache.GetVectorContent(ctx, semverRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(vc.Components).To(Equal(testVectorContent.Components))
		}()

		Eventually(mockReader.vectorCalls.Load).Should(Equal(int64(1)))

		cancelCtx, cancel := context.WithCancel(ctx)
		cancel()
		cancelled := make(chan error, 1)
		go func() {
			_, err := cache.GetVectorContent(cancelCtx, semverRef)
			cancelled <- err
		}()
		Eventually(cancelled).Should(Receive(MatchError(context.Canceled)))

		var waitersWg sync.WaitGroup
		const waiters = 2
		waitersWg.Add(waiters)
		for i := 0; i < waiters; i++ {
			go func() {
				defer waitersWg.Done()
				vc, err := cache.GetVectorContent(ctx, semverRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(vc.Components).To(Equal(testVectorContent.Components))
			}()
		}

		close(mockReader.gate)
		runnerWg.Wait()
		waitersWg.Wait()

		Expect(mockReader.vectorCalls.Load()).To(Equal(int64(1)))
	})

	It("lets a cancelled artifact caller return without blocking the others sharing its fetch", func() {
		mockReader.gate = make(chan struct{})
		cache := newCache()

		var runnerWg sync.WaitGroup
		runnerWg.Add(1)
		go func() {
			defer runnerWg.Done()
			ac, err := cache.GetArtifactContent(ctx, semverRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(ac).To(Equal(testArtifactContent))
		}()

		Eventually(mockReader.artifactCalls.Load).Should(Equal(int64(1)))

		cancelCtx, cancel := context.WithCancel(ctx)
		cancel()
		cancelled := make(chan error, 1)
		go func() {
			_, err := cache.GetArtifactContent(cancelCtx, semverRef)
			cancelled <- err
		}()
		Eventually(cancelled).Should(Receive(MatchError(context.Canceled)))

		var waitersWg sync.WaitGroup
		const waiters = 2
		waitersWg.Add(waiters)
		for i := 0; i < waiters; i++ {
			go func() {
				defer waitersWg.Done()
				ac, err := cache.GetArtifactContent(ctx, semverRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ac).To(Equal(testArtifactContent))
			}()
		}

		close(mockReader.gate)
		runnerWg.Wait()
		waitersWg.Wait()

		Expect(mockReader.artifactCalls.Load()).To(Equal(int64(1)))
	})

	DescribeTable("cached ref entries lifetime",
		func(ct contentType, ref string, advance time.Duration, expectedCalls int64) {
			now := time.Now()
			cache := newCacheAt(&now)
			Expect(get(ctx, ct, cache, ref)).To(Succeed())
			now = now.Add(advance)
			Expect(get(ctx, ct, cache, ref)).To(Succeed())
			Expect(callCount(ct, mockReader)).To(Equal(expectedCalls))
		},
		Entry("semver ref never expires for vectors", vectorContent, semverRef, 100*testTagTTL, int64(1)),
		Entry("semver ref never expires for artifacts", artifactContent, semverRef, 100*testTagTTL, int64(1)),
		Entry("digest-pinned ref never expires for vectors", vectorContent, digestRef, 100*testTagTTL, int64(1)),
		Entry("digest-pinned ref never expires for artifacts", artifactContent, digestRef, 100*testTagTTL, int64(1)),
		Entry("alias ref within the TTL is served from cache for vectors", vectorContent, aliasRef, testTagTTL-time.Second, int64(1)),
		Entry("alias ref within the TTL is served from cache for artifacts", artifactContent, aliasRef, testTagTTL-time.Second, int64(1)),
		Entry("alias ref past the TTL is refetched for vectors", vectorContent, aliasRef, testTagTTL+time.Second, int64(2)),
		Entry("alias ref past the TTL is refetched for artifacts", artifactContent, aliasRef, testTagTTL+time.Second, int64(2)),
	)

	DescribeTable("tagTTL of zero disables caching of floating tags but still caches pinned refs",
		func(ct contentType, ref string, expectedCalls int64) {
			r, err := NewCachingReader(mockReader, testCacheSize, 0)
			Expect(err).ToNot(HaveOccurred())
			cache := r.(*cachingReader)

			Expect(get(ctx, ct, cache, ref)).To(Succeed())
			Expect(get(ctx, ct, cache, ref)).To(Succeed())
			Expect(callCount(ct, mockReader)).To(Equal(expectedCalls))
		},
		Entry("alias ref is refetched every time for vectors", vectorContent, aliasRef, int64(2)),
		Entry("alias ref is refetched every time for artifacts", artifactContent, aliasRef, int64(2)),
		Entry("semver ref is still cached for vectors", vectorContent, semverRef, int64(1)),
		Entry("semver ref is still cached for artifacts", artifactContent, semverRef, int64(1)),
		Entry("digest-pinned ref is still cached for vectors", vectorContent, digestRef, int64(1)),
		Entry("digest-pinned ref is still cached for artifacts", artifactContent, digestRef, int64(1)),
	)

	DescribeTable("inner reader is called again for an entry evicted under size pressure",
		func(ct contentType) {
			r, err := NewCachingReader(mockReader, 2, testTagTTL)
			Expect(err).ToNot(HaveOccurred())
			cache := r.(*cachingReader)

			const (
				refA = refCommonPrefix + "-a:latest"
				refB = refCommonPrefix + "-b:latest"
				refC = refCommonPrefix + "-c:latest"
			)

			Expect(get(ctx, ct, cache, refA)).To(Succeed())
			Expect(get(ctx, ct, cache, refB)).To(Succeed())
			Expect(get(ctx, ct, cache, refC)).To(Succeed())
			Expect(callCount(ct, mockReader)).To(Equal(int64(3)))

			Expect(get(ctx, ct, cache, refA)).To(Succeed())
			Expect(callCount(ct, mockReader)).To(Equal(int64(4)))
		},
		Entry("for vectors", vectorContent),
		Entry("for artifacts", artifactContent),
	)

	It("does not cache failed vector requests", func() {
		mockReader.err = errors.New("registry unreachable")
		cache := newCache()
		_, err := cache.GetVectorContent(ctx, semverRef)
		Expect(err).To(HaveOccurred())
		_, err = cache.GetVectorContent(ctx, semverRef)
		Expect(err).To(HaveOccurred())
		Expect(mockReader.vectorCalls.Load()).To(Equal(int64(2)))
	})

	It("does not cache failed artifact requests", func() {
		mockReader.err = errors.New("registry unreachable")
		cache := newCache()
		_, err := cache.GetArtifactContent(ctx, semverRef)
		Expect(err).To(HaveOccurred())
		_, err = cache.GetArtifactContent(ctx, semverRef)
		Expect(err).To(HaveOccurred())
		Expect(mockReader.artifactCalls.Load()).To(Equal(int64(2)))
	})

	It("returns the bare inner reader when the cache size is zero", func() {
		r, err := NewCachingReader(mockReader, 0, testTagTTL)
		Expect(err).ToNot(HaveOccurred())
		Expect(r).To(BeIdenticalTo(Reader(mockReader)))
	})

	It("returns the bare inner reader when the cache size is negative", func() {
		r, err := NewCachingReader(mockReader, -1, testTagTTL)
		Expect(err).ToNot(HaveOccurred())
		Expect(r).To(BeIdenticalTo(Reader(mockReader)))
	})

})
