package auth

import (
	"fmt"

	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
	"github.com/spf13/cobra"
)

func NewLogoutCmd(appConfig *cfg.AppConfig) (*cobra.Command, error) {
	return &cobra.Command{
		Use:   "logout",
		Short: "Kden API Logout",
		Long:  `Logs the current user out of the Kden API.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			authClient, err := appConfig.APIProvider.AuthClient()
			if err != nil {
				return fmt.Errorf("failed initializing API client: %w", err)
			}

			if err := authClient.Logout(cmd.Context()); err != nil {
				return fmt.Errorf("logging out of Konfidence API failed: %w", err)
			}

			return nil
		},
	}, nil
}
