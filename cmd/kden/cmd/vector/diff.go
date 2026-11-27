package vector

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/konfidence-project/konfidence/internal/kden/apiclient"
	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
	"github.com/konfidence-project/konfidence/internal/kden/output"
	"github.com/spf13/cobra"
)

func NewDiffCmd(appConfig *cfg.AppConfig) (*cobra.Command, error) {
	return &cobra.Command{
		Use:   "diff",
		Short: "Show the diff between two vectors for a given project id",
		Long:  "Retrieve and display the diff of the OCM content between two vectors for a given project id.",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectId, _ := cmd.Flags().GetString("projectId")
			includeUnchanged, _ := cmd.Flags().GetBool("include-unchanged")
			left, _ := cmd.Flags().GetString("left")
			right, _ := cmd.Flags().GetString("right")

			params := &apiclient.GetVectorDiffV1Params{
				Left:  left,
				Right: right,
			}
			if includeUnchanged {
				v := apiclient.True
				params.IncludeUnchanged = &v
			}

			authClient, err := appConfig.APIProvider.AuthClient()
			if err != nil {
				return fmt.Errorf("failed initializing API client: %w", err)
			}

			response, err := authClient.GetVectorDiffV1WithResponse(cmd.Context(), projectId, params)
			if err != nil {
				return fmt.Errorf("getting vector diff failed: %w", err)
			}

			switch response.StatusCode() {
			case http.StatusOK:
				if response.JSON200 == nil {
					return errors.New("vector diff response did not contain a body")
				}
				formatted, err := output.ResolveFormat(response.JSON200, diffTable(response.JSON200))
				if err != nil {
					return fmt.Errorf("formatting vector diff response failed: %w", err)
				}

				output.PrintMessage(formatted)
				return nil

			case http.StatusBadRequest:
				return errors.New(response.JSON400.Error.Message)
			case http.StatusForbidden:
				return errors.New(response.JSON403.Error.Message)
			case http.StatusNotFound:
				return errors.New(response.JSON404.Error.Message)
			default:
				return fmt.Errorf("getting vector diff returned HTTP %d: %s",
					response.StatusCode(),
					string(response.Body),
				)
			}
		},
	}, nil
}
