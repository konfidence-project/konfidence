package upgrade

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/konfidence-project/konfidence/internal/kden/log"
	"github.com/konfidence-project/konfidence/internal/kden/release"
	"github.com/konfidence-project/konfidence/pkg/build"
	"github.com/spf13/cobra"
)

func NewUpgradeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "upgrade",
		Short: "Update the kden CLI to the latest release in place",
		Long: `Download the latest kden release, verify its checksum and atomically
replace the running binary. Use KDEN_VERSION to pin a specific release.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return run(cmd)
		},
	}
}

func run(cmd *cobra.Command) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("self-update is not supported on Windows; download the release archive manually")
	}

	ctx := cmd.Context()
	client := release.New()

	tag := os.Getenv("KDEN_VERSION")
	if tag == "" {
		rel, err := client.Latest(ctx)
		if err != nil {
			return fmt.Errorf("resolving latest release: %w", err)
		}
		tag = rel.Tag
	}

	if tag == build.Version {
		log.Infof("kden is already at %s", build.Version)
		return nil
	}

	log.Infof("upgrading kden from %s to %s", build.Version, tag)
	bin, err := client.DownloadBinary(ctx, tag)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", tag, err)
	}

	if err := replaceExecutable(bin); err != nil {
		return err
	}

	log.Infof("kden upgraded to %s", tag)
	return nil
}

// replaceExecutable atomically swaps the running binary for newBin.
func replaceExecutable(newBin []byte) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locating current executable: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return fmt.Errorf("resolving executable path: %w", err)
	}
	return swapFile(exe, newBin)
}

// swapFile atomically replaces target with newBin. The temp file is created in
// the SAME directory as target so os.Rename is an atomic same-filesystem
// operation; a running process keeps the old inode until it exits.
func swapFile(target string, newBin []byte) error {
	dir := filepath.Dir(target)

	tmp, err := os.CreateTemp(dir, ".kden-upgrade-*")
	if err != nil {
		return fmt.Errorf("creating temp file in %s (need write access to upgrade in place): %w", dir, err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }() // no-op after a successful rename

	if _, err := tmp.Write(newBin); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("writing new binary: %w", err)
	}
	if err := tmp.Chmod(0o755); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod new binary: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing new binary: %w", err)
	}

	if err := os.Rename(tmpName, target); err != nil {
		return fmt.Errorf("replacing %s (need write access to the install dir): %w", target, err)
	}
	return nil
}
