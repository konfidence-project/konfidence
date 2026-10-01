package project

import (
	"errors"
	"fmt"
	"net/http"

	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
	"github.com/konfidence-project/konfidence/internal/kden/output"
	"github.com/spf13/cobra"
)

func NewListCmd(appConfig *cfg.AppConfig) (*cobra.Command, error) {
	return &cobra.Command{
		Use:   "list",
		Short: "List all projects",
		Long:  `Retrieve and display a complete list of all projects.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			authClient, err := appConfig.APIProvider.AuthClient()
			if err != nil {
				return fmt.Errorf("failed initializing API client: %w", err)
			}
			response, err := authClient.ListProjectsV1WithResponse(cmd.Context())
			if err != nil {
				return fmt.Errorf("listing projects failed: %w", err)
			}

			switch response.StatusCode() {
			case http.StatusOK:
				if response.JSON200 == nil {
					return errors.New("projects response did not contain a body")
				}
				formatted, err := output.ResolveFormat(response.JSON200, listTable(response.JSON200))
				if err != nil {
					return fmt.Errorf("formatting projects failed: %w", err)
				}

				output.PrintMessage(formatted)
				return nil

			default:
				return fmt.Errorf(
					"listing projects returned HTTP %d: %s",
					response.StatusCode(),
					string(response.Body),
				)
			}
		},
	}, nil
}
