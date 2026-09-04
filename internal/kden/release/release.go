// Package release resolves, compares and downloads kden CLI releases from
// GitHub. It is the shared seam behind both `kden upgrade` and the background
// update notifier.
package release

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/google/go-github/v89/github"
)

const (
	owner = "konfidence-project"
	repo  = "konfidence"
)

// Release is the minimal view of a GitHub release the CLI cares about.
type Release struct {
	Tag        string
	Prerelease bool
	URL        string // human-facing release page
}

// Client looks up releases. The zero value is usable; it honors GITHUB_TOKEN.
type Client struct {
	gh *github.Client
}

// New returns a Client, authenticating with GITHUB_TOKEN when present to avoid
// unauthenticated rate limits.
func New() *Client {
	var opts []github.ClientOptionsFunc
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		opts = append(opts, github.WithAuthToken(tok))
	}
	// NewClient only errors on bad option config; our options can't produce that.
	gh, _ := github.NewClient(opts...)
	return &Client{gh: gh}
}

// Latest returns the newest release. It first tries GitHub's "latest" endpoint
// (which excludes prereleases and 404s on prerelease-only repos) and falls back
// to the most recent of all releases so prerelease-only projects still resolve.
func (c *Client) Latest(ctx context.Context) (*Release, error) {
	rel, resp, err := c.gh.Repositories.GetLatestRelease(ctx, owner, repo)
	if err == nil {
		return toRelease(rel), nil
	}
	if resp == nil || resp.StatusCode != http.StatusNotFound {
		return nil, fmt.Errorf("looking up latest release: %w", err)
	}

	// No non-prerelease published yet: take the newest release overall.
	rels, _, err := c.gh.Repositories.ListReleases(ctx, owner, repo, &github.ListOptions{PerPage: 1})
	if err != nil {
		return nil, fmt.Errorf("listing releases: %w", err)
	}
	if len(rels) == 0 {
		return nil, fmt.Errorf("no releases published for %s/%s", owner, repo)
	}
	return toRelease(rels[0]), nil
}

func toRelease(r *github.RepositoryRelease) *Release {
	return &Release{
		Tag:        r.GetTagName(),
		Prerelease: r.GetPrerelease(),
		URL:        r.GetHTMLURL(),
	}
}

// Newer reports whether latest is a strictly higher semver than current.
// Non-semver inputs (e.g. "dev", a git ref, a sha) return false: we never
// prompt someone running a dev or source build to "upgrade".
func Newer(current, latest string) bool {
	cur, err := semverParse(current)
	if err != nil {
		return false
	}
	lat, err := semverParse(latest)
	if err != nil {
		return false
	}
	return lat.GreaterThan(cur)
}

// semverParse parses a version, tolerating an optional leading "v".
func semverParse(v string) (*semver.Version, error) {
	return semver.NewVersion(strings.TrimPrefix(v, "v"))
}

// ArchiveName mirrors the archive name_template in .goreleaser.yaml:
//
//	kden-cli-<os>-<arch>.tar.gz   (amd64 -> x86_64, arm64 stays arm64)
//
// A mismatch here is a 404 for every user, so it is covered by a table test.
func ArchiveName(goos, goarch string) string {
	arch := goarch
	if arch == "amd64" {
		arch = "x86_64"
	}
	return fmt.Sprintf("kden-cli-%s-%s.tar.gz", goos, arch)
}

// ArchiveNameForHost returns the archive name for the running platform.
func ArchiveNameForHost() string {
	return ArchiveName(runtime.GOOS, runtime.GOARCH)
}
