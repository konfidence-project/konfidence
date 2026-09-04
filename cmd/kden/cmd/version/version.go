package version

import (
	"fmt"
	"runtime"
	"strings"

	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
	"github.com/konfidence-project/konfidence/internal/kden/output"
	"github.com/konfidence-project/konfidence/pkg/build"
	"github.com/spf13/cobra"
)

// devVersion is the ldflags default in pkg/build; a binary reporting it was not
// built from a release, so there is no newer version to point an update at.
const devVersion = "dev"

// installCommand is the one-liner users re-run to update. Updating is a
// deliberate act (never automatic): the CLI is a client of the API/controllers
// and must not silently move ahead of the server it talks to.
const installCommand = "curl -fsSL https://konfidence.cloud/install.sh | sh"

// Info is the machine-readable version payload. Tags keep json and yaml output
// lowercase and stable for scripts.
type Info struct {
	Version   string `json:"version" yaml:"version"`
	Commit    string `json:"commit" yaml:"commit"`
	GoVersion string `json:"goVersion" yaml:"goVersion"`
	Platform  string `json:"platform" yaml:"platform"`
	BuildDate string `json:"buildDate" yaml:"buildDate"`
}

func current() Info {
	return Info{
		Version:   build.Version,
		Commit:    build.GitCommit,
		GoVersion: runtime.Version(),
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
		BuildDate: build.Date,
	}
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the kden CLI version",
	Long: `Print the kden CLI version, build metadata and platform.

Respects the global --output flag (json, yaml, pretty). When run against a
released build, prints the command to re-run to update (to stderr, so it never
pollutes json/yaml/pretty output).`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		info := current()

		switch output.FormattedOutput(strings.ToLower(cfg.Config.Output)) {
		case output.TablePrettyOutputFormat:
			writePretty(cmd, info)
		default:
			// json / yaml (and any other configured format) go through the
			// shared formatter so version matches every other command.
			formatted, err := output.ResolveFormat(info, "version")
			if err != nil {
				return fmt.Errorf("formatting version failed: %w", err)
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), strings.TrimRight(formatted, "\n"))
		}

		// Update hint on stderr, released builds only.
		if info.Version != devVersion {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(),
				"\nTo update, re-run the installer:\n  %s\n", installCommand)
		}
		return nil
	},
}

// writePretty renders an aligned, human-readable block. A version is scalar, not
// tabular, so this is written directly rather than through the list table model.
func writePretty(cmd *cobra.Command, info Info) {
	w := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(w, "kden %s\n", info.Version)
	_, _ = fmt.Fprintf(w, "  Commit:     %s\n", info.Commit)
	_, _ = fmt.Fprintf(w, "  Go:         %s\n", info.GoVersion)
	_, _ = fmt.Fprintf(w, "  Platform:   %s\n", info.Platform)
	_, _ = fmt.Fprintf(w, "  Built:      %s\n", info.BuildDate)
}

func NewVersionCmd() *cobra.Command {
	return versionCmd
}
