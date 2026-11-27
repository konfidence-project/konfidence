package content

import (
	"context"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	kompref "github.com/konfidence-project/konfidence/pkg/ocm/compref"
	"golang.org/x/sync/singleflight"
)

type vectorEntry struct {
	val       VectorContent
	expiresAt time.Time
}

type artifactEntry struct {
	val       ArtifactContent
	expiresAt time.Time
}

type cachingReader struct {
	inner         Reader
	vectorCache   *lru.Cache[string, vectorEntry]
	artifactCache *lru.Cache[string, artifactEntry]
	vectorGroup   singleflight.Group
	artifactGroup singleflight.Group
	tagTTL        time.Duration
	clock         func() time.Time
}

func NewCachingReader(inner Reader, size int, tagTTL time.Duration) (Reader, error) {
	if size <= 0 {
		return inner, nil
	}
	vectorCache, err := lru.New[string, vectorEntry](size)
	if err != nil {
		return nil, err
	}
	artifactCache, err := lru.New[string, artifactEntry](size)
	if err != nil {
		return nil, err
	}
	return &cachingReader{
		inner:         inner,
		vectorCache:   vectorCache,
		artifactCache: artifactCache,
		tagTTL:        tagTTL,
		clock:         time.Now,
	}, nil
}

func (c *cachingReader) GetVectorContent(ctx context.Context, rawOcmRef string) (VectorContent, error) {
	if entry, ok := c.vectorCache.Get(rawOcmRef); ok && !c.isExpired(entry.expiresAt) {
		return entry.val, nil
	}

	ch := c.vectorGroup.DoChan(rawOcmRef, func() (any, error) {
		fetchCtx := context.WithoutCancel(ctx)
		val, err := c.inner.GetVectorContent(fetchCtx, rawOcmRef)
		if err != nil {
			return VectorContent{}, err
		}
		if expiresAt, shouldCache := c.cacheDecision(rawOcmRef); shouldCache {
			c.vectorCache.Add(rawOcmRef, vectorEntry{val: val, expiresAt: expiresAt})
		}
		return val, nil
	})

	select {
	case res := <-ch:
		if res.Err != nil {
			return VectorContent{}, res.Err
		}
		return res.Val.(VectorContent), nil
	case <-ctx.Done():
		return VectorContent{}, ctx.Err()
	}
}

func (c *cachingReader) GetArtifactContent(ctx context.Context, rawOcmRef string) (ArtifactContent, error) {
	if entry, ok := c.artifactCache.Get(rawOcmRef); ok && !c.isExpired(entry.expiresAt) {
		return entry.val, nil
	}

	ch := c.artifactGroup.DoChan(rawOcmRef, func() (any, error) {
		fetchCtx := context.WithoutCancel(ctx)
		val, err := c.inner.GetArtifactContent(fetchCtx, rawOcmRef)
		if err != nil {
			return ArtifactContent{}, err
		}
		if expiresAt, shouldCache := c.cacheDecision(rawOcmRef); shouldCache {
			c.artifactCache.Add(rawOcmRef, artifactEntry{val: val, expiresAt: expiresAt})
		}
		return val, nil
	})

	select {
	case res := <-ch:
		if res.Err != nil {
			return ArtifactContent{}, res.Err
		}
		return res.Val.(ArtifactContent), nil
	case <-ctx.Done():
		return ArtifactContent{}, ctx.Err()
	}
}

func (c *cachingReader) isExpired(expiresAt time.Time) bool {
	return !expiresAt.IsZero() && expiresAt.Before(c.clock())
}

func (c *cachingReader) cacheDecision(rawOcmRef string) (expiresAt time.Time, shouldCache bool) {
	parsed, err := kompref.Parse(rawOcmRef)
	if err == nil && parsed != nil && (parsed.Digest != "" || kompref.IsSemver(parsed.Version)) {
		return time.Time{}, true
	}
	if c.tagTTL <= 0 {
		return time.Time{}, false
	}
	return c.clock().Add(c.tagTTL), true
}
