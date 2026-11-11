//nolint:dupl // Keep action-specific command setup explicit; response handling is shared.
package vectorpromotion

import (
	"fmt"

	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
	"github.com/konfidence-project/konfidence/internal/kden/output"
	"github.com/spf13/cobra"
)

func NewRevokeApprovalCmd(appConfig *cfg.AppConfig) (*cobra.Command, error) {
	return &cobra.Command{
		Use:   "revoke-approval <vectorPromotionId>",
		Short: "Revoke the approval of a vector promotion for a specific project id",
		Long: "Set the explicit-approval.konfidence.cloud gate of the waiting vector promotion with the given " +
			"vector promotion id back to Pending.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectId, _ := cmd.Flags().GetString("projectId")
			vectorPromotionId := args[0]
			authClient, err := appConfig.APIProvider.AuthClient()
			if err != nil {
				return fmt.Errorf("failed initializing API client: %w", err)
			}

			response, err := authClient.RevokeApprovalVectorPromotionV1WithResponse(
				cmd.Context(), projectId, vectorPromotionId,
			)
			if err != nil {
				return fmt.Errorf("revoking vector promotion approval failed: %w", err)
			}
			if err := checkActionResponse(response.StatusCode(), response.Body, "revoking vector promotion approval"); err != nil {
				return err
			}

			output.PrintMessage(fmt.Sprintf("Approval of vector promotion %s revoked successfully", vectorPromotionId))
			return nil
		},
	}, nil
}
