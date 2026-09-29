package landscape

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
		Short: "List landscapes for a given project id",
		Long:  `Retrieve and display all landscapes for a given project id.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			projectId, _ := cmd.Flags().GetString("projectId")
			authClient, err := appConfig.APIProvider.AuthClient()
			if err != nil {
				return fmt.Errorf("failed initializing API client: %w", err)
			}

			response, err := authClient.ListLandscapesV1WithResponse(cmd.Context(), projectId)
			if err != nil {
				return fmt.Errorf("listing landscapes failed: %w", err)
			}

			switch response.StatusCode() {
			case http.StatusOK:
				if response.JSON200 == nil {
					return errors.New("landscapes response did not contain a body")
				}
				formatted, err := output.ResolveFormat(response.JSON200, "landscape-list")
				if err != nil {
					return fmt.Errorf("formatting landscapes failed: %w", err)
				}

				output.PrintMessage(formatted)
				return nil

			case http.StatusForbidden:
				return errors.New(response.JSON403.Error.Message)

			case http.StatusNotFound:
				return errors.New(response.JSON404.Error.Message)

			default:
				return fmt.Errorf(
					"listing landscapes returned HTTP %d: %s",
					response.StatusCode(),
					string(response.Body),
				)
			}
		},
	}, nil
}
