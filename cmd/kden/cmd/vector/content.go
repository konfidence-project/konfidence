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

func NewContentCmd(appConfig *cfg.AppConfig) (*cobra.Command, error) {
	return &cobra.Command{
		Use:   "content",
		Short: "List OCM content of vectors for a given project id",
		Long:  "Retrieve and display the OCM content of one or more vectors for a given project id.",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectId, _ := cmd.Flags().GetString("projectId")
			refs, _ := cmd.Flags().GetStringSlice("refs")

			params := &apiclient.GetVectorContentV1Params{Refs: refs}

			authClient, err := appConfig.APIProvider.AuthClient()
			if err != nil {
				return fmt.Errorf("failed initializing API client: %w", err)
			}

			response, err := authClient.GetVectorContentV1WithResponse(cmd.Context(), projectId, params)
			if err != nil {
				return fmt.Errorf("getting vector content failed: %w", err)
			}

			switch response.StatusCode() {
			case http.StatusOK:
				if response.JSON200 == nil {
					return errors.New("vector content response did not contain a body")
				}
				formatted, err := output.ResolveFormat(response.JSON200, contentTable(response.JSON200))
				if err != nil {
					return fmt.Errorf("formatting vector content response failed: %w", err)
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
				return fmt.Errorf("getting vector content returned HTTP %d: %s",
					response.StatusCode(),
					string(response.Body),
				)
			}
		},
	}, nil
}
