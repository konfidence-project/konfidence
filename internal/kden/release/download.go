package release

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const downloadTimeout = 60 * time.Second

// DownloadBinary fetches the kden archive for the given tag, verifies its
// sha256 against the release's checksums.txt, extracts the kden binary and
// returns its bytes. The trust anchor is the checksum: a mismatch is fatal.
func (c *Client) DownloadBinary(ctx context.Context, tag string) ([]byte, error) {
	asset := ArchiveNameForHost()
	base := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s", owner, repo, tag)

	archive, err := c.httpGet(ctx, base+"/"+asset)
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", asset, err)
	}
	sums, err := c.httpGet(ctx, base+"/checksums.txt")
	if err != nil {
		return nil, fmt.Errorf("downloading checksums.txt: %w", err)
	}

	if err := verifySHA256(archive, sums, asset); err != nil {
		return nil, err
	}
	return extractKden(archive)
}

func (c *Client) httpGet(ctx context.Context, url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, downloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// verifySHA256 checks archive's digest against the "<hex>  <name>" line for
// asset in a goreleaser checksums.txt.
func verifySHA256(archive, checksums []byte, asset string) error {
	sum := sha256.Sum256(archive)
	got := hex.EncodeToString(sum[:])

	var want string
	sc := bufio.NewScanner(bytes.NewReader(checksums))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) == 2 && fields[1] == asset {
			want = fields[0]
			break
		}
	}
	if want == "" {
		return fmt.Errorf("no checksum for %s in checksums.txt", asset)
	}
	if !strings.EqualFold(want, got) {
		return fmt.Errorf("checksum mismatch for %s: want %s, got %s", asset, want, got)
	}
	return nil
}

// extractKden pulls the "kden" binary out of a .tar.gz archive.
func extractKden(targz []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(targz))
	if err != nil {
		return nil, fmt.Errorf("opening gzip: %w", err)
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading tar: %w", err)
		}
		if hdr.Typeflag == tar.TypeReg && (hdr.Name == "kden" || strings.HasSuffix(hdr.Name, "/kden")) {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("kden binary not found in archive")
}
