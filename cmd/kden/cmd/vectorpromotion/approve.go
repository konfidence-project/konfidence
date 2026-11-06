package vectorpromotion

import (
	"fmt"

	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
	"github.com/spf13/cobra"
)

func NewApproveCmd(appConfig *cfg.AppConfig) (*cobra.Command, error) {
	return &cobra.Command{
		Use:   "approve <vectorPromotionId>",
		Short: "Approve a vector promotion for a specific project id",
		Long: "Satisfy the explicit-approval.konfidence.cloud gate of the vector promotion with the given " +
			"vector promotion id for a specific project id. The promotion must declare that gate.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectId, _ := cmd.Flags().GetString("projectId")
			vectorPromotionId := args[0]
			authClient, err := appConfig.APIProvider.AuthClient()
			if err != nil {
				return fmt.Errorf("failed initializing API client: %w", err)
			}

			response, err := authClient.ApproveVectorPromotionV1WithResponse(cmd.Context(), projectId, vectorPromotionId)
			if err != nil {
				return fmt.Errorf("approving vector promotion failed: %w", err)
			}

			return handleApprovalActionResponse(
				response,
				fmt.Sprintf("Vector promotion %s approved successfully", vectorPromotionId),
				"approving vector promotion",
			)
		},
	}, nil
}
