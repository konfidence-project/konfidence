package vectorpromotion

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/konfidence-project/konfidence/cmd/kden/cmd/auth"
	"github.com/konfidence-project/konfidence/internal/kden/apiclient"
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

			response, err := auth.RequestWithAuthRetry(cmd.Context(), authClient,
				func(ctx context.Context) (*apiclient.ListVectorPromotionConfigsV1Response, error) {
					return authClient.KdenApiClient().ListVectorPromotionConfigsV1WithResponse(ctx, projectId)
				})
			if err != nil {
				return fmt.Errorf("listing vector promotion configs failed: %w", err)
			}

			switch response.StatusCode() {
			case http.StatusOK:
				if response.JSON200 == nil {
					return errors.New("vector promotion configs response did not contain a body")
				}
				formatted, err := output.ResolveFormat(response.JSON200, "vector-promotion-config-list")
				if err != nil {
					return fmt.Errorf("formatting vector promotion configs failed: %w", err)
				}

				output.PrintMessage(formatted)
				return nil

			case http.StatusUnauthorized:
				return errors.New("api rejected the newly established session")

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
