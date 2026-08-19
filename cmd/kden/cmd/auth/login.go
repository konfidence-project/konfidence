package auth

import (
	"fmt"

	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
	"github.com/spf13/cobra"
)

func NewLoginCmd(appConfig *cfg.AppConfig) (*cobra.Command, error) {
	return &cobra.Command{
		Use:   "login",
		Short: "Kden API Login",
		Long:  `Start the login process for the Kden API.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			authClient, err := appConfig.APIProvider.AuthClient()
			if err != nil {
				return fmt.Errorf("failed initializing API client: %w", err)
			}

			if err := authClient.Login(cmd.Context()); err != nil {
				return fmt.Errorf("authenticating with Konfidence API failed: %w", err)
			}

			return nil
		},
	}, nil
}
