package vectorpromotion

import (
	"fmt"

	"github.com/konfidence-project/konfidence/internal/kden/apiclient"
	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
	"github.com/konfidence-project/konfidence/internal/kden/output"
	"github.com/spf13/cobra"
)

const reasonFlag = "reason"

//nolint:dupl // Keep action-specific command setup explicit; response handling is shared.
func NewRejectCmd(appConfig *cfg.AppConfig) (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:   "reject <vectorPromotionId>",
		Short: "Reject a vector promotion for a specific project id",
		Long: "Terminally reject the waiting vector promotion with the given vector promotion id for a specific " +
			"project id, so it never executes, regardless of the state of its gates. Only promotions still waiting " +
			"on their gates can be rejected. The gate states are kept as reported, and you and the given reason " +
			"are recorded on the promotion.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectId, _ := cmd.Flags().GetString("projectId")
			reason, _ := cmd.Flags().GetString(reasonFlag)
			vectorPromotionId := args[0]
			authClient, err := appConfig.APIProvider.AuthClient()
			if err != nil {
				return fmt.Errorf("failed initializing API client: %w", err)
			}

			response, err := authClient.RejectVectorPromotionV1WithResponse(cmd.Context(), projectId, vectorPromotionId,
				apiclient.RejectVectorPromotionV1JSONRequestBody{Reason: reason})
			if err != nil {
				return fmt.Errorf("rejecting vector promotion failed: %w", err)
			}
			if err := checkActionResponse(response.StatusCode(), response.Body, "rejecting vector promotion"); err != nil {
				return err
			}

			output.PrintMessage(fmt.Sprintf("Vector promotion %s rejected successfully", vectorPromotionId))
			return nil
		},
	}
	cmd.Flags().StringP(reasonFlag, "r", "", "Why the promotion is rejected (required)")
	if err := cmd.MarkFlagRequired(reasonFlag); err != nil {
		panic(err)
	}
	return cmd, nil
}

//nolint:dupl // Keep action-specific command setup explicit; response handling is shared.
func NewBypassGatesCmd(appConfig *cfg.AppConfig) (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:   "bypass-gates <vectorPromotionId>",
		Short: "Bypass the gates of a vector promotion for a specific project id",
		Long: "Let the waiting vector promotion with the given vector promotion id for a specific project id " +
			"proceed and execute as if all of its gates were satisfied. Only promotions still waiting on their " +
			"gates can be bypassed. The gate states are kept as reported, and you and the given reason are " +
			"recorded on the promotion.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectId, _ := cmd.Flags().GetString("projectId")
			reason, _ := cmd.Flags().GetString(reasonFlag)
			vectorPromotionId := args[0]
			authClient, err := appConfig.APIProvider.AuthClient()
			if err != nil {
				return fmt.Errorf("failed initializing API client: %w", err)
			}

			response, err := authClient.BypassVectorPromotionGatesV1WithResponse(cmd.Context(), projectId, vectorPromotionId,
				apiclient.BypassVectorPromotionGatesV1JSONRequestBody{Reason: reason})
			if err != nil {
				return fmt.Errorf("bypassing vector promotion gates failed: %w", err)
			}
			if err := checkActionResponse(response.StatusCode(), response.Body, "bypassing vector promotion gates"); err != nil {
				return err
			}

			output.PrintMessage(fmt.Sprintf("Gates of vector promotion %s bypassed successfully", vectorPromotionId))
			return nil
		},
	}
	cmd.Flags().StringP(reasonFlag, "r", "", "Why the gates are bypassed (required)")
	if err := cmd.MarkFlagRequired(reasonFlag); err != nil {
		panic(err)
	}
	return cmd, nil
}
