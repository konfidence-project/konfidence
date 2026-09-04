package upgrade

import (
	"os"
	"path/filepath"
	"testing"
)

// replaceExecutable swaps os.Executable(), which in a test IS the test binary —
// we must not clobber it. So exercise the atomic-swap core against a fake path.
func TestAtomicSwap(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "kden")
	if err := os.WriteFile(target, []byte("OLD"), 0o755); err != nil {
		t.Fatal(err)
	}

	newBin := []byte("NEWBINARY")
	if err := swapFile(target, newBin); err != nil {
		t.Fatalf("swapFile: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(newBin) {
		t.Errorf("after swap got %q, want %q", got, newBin)
	}
	info, _ := os.Stat(target)
	if info.Mode().Perm()&0o100 == 0 {
		t.Errorf("replaced binary is not executable: %v", info.Mode())
	}
}
