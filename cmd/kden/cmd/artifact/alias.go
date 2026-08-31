package artifact

import (
	"fmt"

	"github.com/konfidence-project/konfidence/internal/kden/ocm"
	"github.com/spf13/cobra"
)

var Alias = ocm.Alias

func NewAliasCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alias <source-ref> <alias>",
		Short: "Create or update a mutable alias tag for a component version",
		Long: `Create or update a mutable alias tag pointing to an existing component version.

    kden artifact alias registry.example.com//konfidence.io/payment-hub:1.0.0 edge`,
		Args: cobra.ExactArgs(2),
		RunE: runAlias,
	}
	return cmd
}

func runAlias(cmd *cobra.Command, args []string) error {
	ocmConfig, err := GetOcmConfiguration(cmd)
	if err != nil {
		return err
	}

	if err := Alias(cmd.Context(), ocm.AliasProperties{
		ComponentVersion: args[0],
		Alias:            args[1],
	}, ocmConfig); err != nil {
		return fmt.Errorf("failed to create alias %q for %s: %w", args[1], args[0], err)
	}

	return nil
}
