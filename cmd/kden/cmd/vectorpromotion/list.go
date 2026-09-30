package vectorpromotion

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
		Short: "List vector promotion configs for a given project id",
		Long:  `Retrieve and display all vector promotion configs for a given project id.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			projectId, _ := cmd.Flags().GetString("projectId")
			authClient, err := appConfig.APIProvider.AuthClient()
			if err != nil {
				return fmt.Errorf("failed initializing API client: %w", err)
			}

			response, err := authClient.ListVectorPromotionConfigsV1WithResponse(cmd.Context(), projectId)
			if err != nil {
				return fmt.Errorf("listing vector promotion configs failed: %w", err)
			}

			switch response.StatusCode() {
			case http.StatusOK:
				if response.JSON200 == nil {
					return errors.New("vector promotion configs response did not contain a body")
				}
				formatted, err := output.ResolveFormat(response.JSON200, configListTable(response.JSON200))
				if err != nil {
					return fmt.Errorf("formatting vector promotion configs failed: %w", err)
				}

				output.PrintMessage(formatted)
				return nil

			case http.StatusForbidden:
				return errors.New(response.JSON403.Error.Message)

			case http.StatusNotFound:
				return errors.New(response.JSON404.Error.Message)

			default:
				return fmt.Errorf(
					"listing vector promotion configs returned HTTP %d: %s",
					response.StatusCode(),
					string(response.Body),
				)
			}
		},
	}, nil
}
