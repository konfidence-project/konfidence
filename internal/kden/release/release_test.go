package release

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestArchiveName(t *testing.T) {
	// Must mirror .goreleaser.yaml name_template exactly, or every download 404s.
	cases := []struct{ goos, goarch, want string }{
		{"linux", "amd64", "kden-cli-linux-x86_64.tar.gz"},
		{"linux", "arm64", "kden-cli-linux-arm64.tar.gz"},
		{"darwin", "arm64", "kden-cli-darwin-arm64.tar.gz"},
		{"darwin", "amd64", "kden-cli-darwin-x86_64.tar.gz"},
	}
	for _, c := range cases {
		if got := ArchiveName(c.goos, c.goarch); got != c.want {
			t.Errorf("ArchiveName(%q,%q)=%q, want %q", c.goos, c.goarch, got, c.want)
		}
	}
}

func TestNewer(t *testing.T) {
	cases := []struct {
		current, latest string
		want            bool
	}{
		{"v0.3.0", "v0.4.0", true},
		{"0.3.0", "0.4.0", true},
		{"v0.4.0", "v0.4.0", false},
		{"v0.5.0", "v0.4.0", false},
		{"dev", "v0.4.0", false},  // dev build: never nag
		{"v0.3.0", "main", false}, // non-semver latest
		{"v1.0.0", "v1.0.1", true},
	}
	for _, c := range cases {
		if got := Newer(c.current, c.latest); got != c.want {
			t.Errorf("Newer(%q,%q)=%v, want %v", c.current, c.latest, got, c.want)
		}
	}
}

func TestVerifySHA256(t *testing.T) {
	archive := []byte("kden-payload")
	sum := sha256.Sum256(archive)
	hexsum := hex.EncodeToString(sum[:])
	asset := "kden-cli-linux-x86_64.tar.gz"
	checksums := []byte(hexsum + "  " + asset + "\ndeadbeef  other.tar.gz\n")

	if err := verifySHA256(archive, checksums, asset); err != nil {
		t.Errorf("valid checksum rejected: %v", err)
	}
	if err := verifySHA256([]byte("tampered"), checksums, asset); err == nil {
		t.Error("mismatched checksum accepted")
	}
	if err := verifySHA256(archive, checksums, "missing.tar.gz"); err == nil {
		t.Error("missing checksum entry accepted")
	}
}

func TestExtractKden(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	payload := []byte("#!binary")
	_ = tw.WriteHeader(&tar.Header{Name: "kden", Mode: 0o755, Size: int64(len(payload)), Typeflag: tar.TypeReg})
	_, _ = tw.Write(payload)
	_ = tw.Close()
	_ = gz.Close()

	got, err := extractKden(buf.Bytes())
	if err != nil {
		t.Fatalf("extractKden: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("extracted %q, want %q", got, payload)
	}

	if _, err := extractKden([]byte("not a gzip")); err == nil {
		t.Error("expected error on non-gzip input")
	}
}
