package vectorpromotion

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/konfidence-project/konfidence/internal/kden/apiclient"
	kdenauth "github.com/konfidence-project/konfidence/internal/kden/auth"
	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
	"github.com/konfidence-project/konfidence/internal/kden/output"
	"github.com/spf13/cobra"
)

const reasonFlag = "reason"

func NewRejectCmd(appConfig *cfg.AppConfig) (*cobra.Command, error) {
	return newOverrideCmd(appConfig, overrideCmdSpec{
		use:   "reject <vectorPromotionId>",
		short: "Reject a vector promotion for a specific project id",
		long: "Terminally reject the waiting vector promotion with the given vector promotion id for a specific " +
			"project id, so it never executes, regardless of the state of its gates. Only promotions still waiting " +
			"on their gates can be rejected. The gate states are kept as reported, and you and the given reason " +
			"are recorded on the promotion.",
		reasonUsage: "Why the promotion is rejected (required)",
		action:      "rejecting vector promotion",
		success:     "Vector promotion %s rejected successfully",
		send: func(ctx context.Context, c *kdenauth.Client, projectId, vectorPromotionId, reason string) (overrideResponse, error) {
			response, err := c.RejectVectorPromotionV1WithResponse(ctx, projectId, vectorPromotionId,
				apiclient.RejectVectorPromotionV1JSONRequestBody{Reason: reason})
			if err != nil {
				return overrideResponse{}, err
			}
			return overrideResponse{
				status: response.StatusCode(), body: response.Body,
				json400: response.JSON400, json403: response.JSON403, json404: response.JSON404, json409: response.JSON409,
			}, nil
		},
	}), nil
}

func NewBypassGatesCmd(appConfig *cfg.AppConfig) (*cobra.Command, error) {
	return newOverrideCmd(appConfig, overrideCmdSpec{
		use:   "bypass-gates <vectorPromotionId>",
		short: "Bypass the gates of a vector promotion for a specific project id",
		long: "Let the waiting vector promotion with the given vector promotion id for a specific project id " +
			"proceed and execute as if all of its gates were satisfied. Only promotions still waiting on their " +
			"gates can be bypassed. The gate states are kept as reported, and you and the given reason are " +
			"recorded on the promotion.",
		reasonUsage: "Why the gates are bypassed (required)",
		action:      "bypassing vector promotion gates",
		success:     "Gates of vector promotion %s bypassed successfully",
		send: func(ctx context.Context, c *kdenauth.Client, projectId, vectorPromotionId, reason string) (overrideResponse, error) {
			response, err := c.BypassVectorPromotionGatesV1WithResponse(ctx, projectId, vectorPromotionId,
				apiclient.BypassVectorPromotionGatesV1JSONRequestBody{Reason: reason})
			if err != nil {
				return overrideResponse{}, err
			}
			return overrideResponse{
				status: response.StatusCode(), body: response.Body,
				json400: response.JSON400, json403: response.JSON403, json404: response.JSON404, json409: response.JSON409,
			}, nil
		},
	}), nil
}

type overrideCmdSpec struct {
	use, short, long string
	reasonUsage      string
	action           string
	// success is a format string taking the promotion id.
	success string
	send    func(ctx context.Context, c *kdenauth.Client, projectId, vectorPromotionId, reason string) (overrideResponse, error)
}

type overrideResponse struct {
	status                             int
	body                               []byte
	json400, json403, json404, json409 *apiclient.ErrorResponse
}

func newOverrideCmd(appConfig *cfg.AppConfig, spec overrideCmdSpec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   spec.use,
		Short: spec.short,
		Long:  spec.long,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectId, _ := cmd.Flags().GetString("projectId")
			reason, _ := cmd.Flags().GetString(reasonFlag)
			vectorPromotionId := args[0]
			authClient, err := appConfig.APIProvider.AuthClient()
			if err != nil {
				return fmt.Errorf("failed initializing API client: %w", err)
			}

			response, err := spec.send(cmd.Context(), authClient, projectId, vectorPromotionId, reason)
			if err != nil {
				return fmt.Errorf("%s failed: %w", spec.action, err)
			}

			var apiErr *apiclient.ErrorResponse
			switch response.status {
			case http.StatusNoContent:
				output.PrintMessage(fmt.Sprintf(spec.success, vectorPromotionId))
				return nil
			case http.StatusBadRequest:
				apiErr = response.json400
			case http.StatusForbidden:
				apiErr = response.json403
			case http.StatusNotFound:
				apiErr = response.json404
			case http.StatusConflict:
				apiErr = response.json409
			}
			if apiErr != nil {
				return errors.New(apiErr.Error.Message)
			}
			return fmt.Errorf("%s returned HTTP %d: %s", spec.action, response.status, string(response.body))
		},
	}
	cmd.Flags().StringP(reasonFlag, "r", "", spec.reasonUsage)
	if err := cmd.MarkFlagRequired(reasonFlag); err != nil {
		panic(err)
	}
	return cmd
}
