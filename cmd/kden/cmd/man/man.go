package man

import (
	"fmt"
	"os"

	"github.com/konfidence-project/konfidence/internal/kden/log"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var docsCmd = &cobra.Command{
	Use:    "docs",
	Short:  "Generate documentation",
	Hidden: true, // Hide from regular help
}

var manCmd = &cobra.Command{
	Use:   "man [output-dir]",
	Short: "Generate local man pages",
	Long: `Generate local man pages for kden and all subcommands.

The man pages are generated in the specified output directory.
Default output directory is ./man/

Examples:
  kden docs man
  kden docs man /usr/local/share/man/man1`,

	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		outputDir := "./man"
		if len(args) > 0 {
			outputDir = args[0]
		}

		// Create output directory if it doesn't exist
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		header := &doc.GenManHeader{
			Title:   "KDEN",
			Section: "1",
			Source:  "kden",
			Manual:  "User Commands",
		}

		if err := doc.GenManTree(cmd.Root(), header, outputDir); err != nil {
			return fmt.Errorf("failed to generate man pages: %w", err)
		}

		log.Infof("Man pages generated in %s", outputDir)
		return nil
	},
}

func NewManCmd() *cobra.Command {
	docsCmd.AddCommand(manCmd)
	return docsCmd
}
