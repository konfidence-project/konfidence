package vectorpromotion

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/konfidence-project/konfidence/cmd/kden/cmd/auth"
	"github.com/konfidence-project/konfidence/internal/kden/apiclient"
	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
	"github.com/spf13/cobra"
)

func NewApproveCmd(appConfig *cfg.AppConfig) (*cobra.Command, error) {
	return &cobra.Command{
		Use:   "approve <vectorPromotionId>",
		Short: "Approve a vector promotion for a given project id",
		Long:  "Grant approval for a vector promotion with the given vector promotion id for a given project id.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectId, _ := cmd.Flags().GetString("projectId")
			vectorPromotionId := args[0]
			authClient, err := appConfig.APIProvider.AuthClient()
			if err != nil {
				return fmt.Errorf("failed initializing API client: %w", err)
			}

			response, err := auth.RequestWithAuthRetry(cmd.Context(), authClient,
				func(ctx context.Context) (*apiclient.ApproveVectorPromotionV1Response, error) {
					return authClient.KdenApiClient().ApproveVectorPromotionV1WithResponse(ctx, projectId, vectorPromotionId)
				})
			if err != nil {
				return fmt.Errorf("approving vector promotion failed: %w", err)
			}

			switch response.StatusCode() {
			case http.StatusNoContent:
				return nil

			case http.StatusUnauthorized:
				return errors.New("api rejected the newly established session")

			case http.StatusForbidden:
				return errors.New(response.JSON403.Error.Message)

			case http.StatusNotFound:
				return errors.New(response.JSON404.Error.Message)

			case http.StatusConflict:
				return errors.New(response.JSON409.Error.Message)

			default:
				return fmt.Errorf(
					"approving vector promotion returned HTTP %d: %s",
					response.StatusCode(),
					string(response.Body),
				)
			}
		},
	}, nil
}
