package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestLoadOIDCClientSecret(t *testing.T) {
	t.Run("loads secret from environment", func(t *testing.T) {
		t.Setenv("API_OIDC_CLIENT_SECRET", "environment-secret")
		cfg.OIDC.ClientSecret = ""
		cmd := &cobra.Command{}
		cmd.Flags().String("oidc-client-secret", "", "")

		loadOIDCClientSecret(cmd, nil)

		if cfg.OIDC.ClientSecret != "environment-secret" {
			t.Fatalf("expected environment secret, got %q", cfg.OIDC.ClientSecret)
		}
	})

	t.Run("preserves explicit flag value", func(t *testing.T) {
		t.Setenv("API_OIDC_CLIENT_SECRET", "environment-secret")
		cfg.OIDC.ClientSecret = "flag-secret"
		cmd := &cobra.Command{}
		cmd.Flags().String("oidc-client-secret", "", "")
		if err := cmd.Flags().Set("oidc-client-secret", "flag-secret"); err != nil {
			t.Fatal(err)
		}

		loadOIDCClientSecret(cmd, nil)

		if cfg.OIDC.ClientSecret != "flag-secret" {
			t.Fatalf("expected flag secret, got %q", cfg.OIDC.ClientSecret)
		}
	})
}

func TestEnvList(t *testing.T) {
	t.Setenv("API_OIDC_ALLOW_RETURN_URLS", "https://one.example.com/callback,https://two.example.com/login")

	values := envList("API_OIDC_ALLOW_RETURN_URLS")
	if len(values) != 2 || values[0] != "https://one.example.com/callback" || values[1] != "https://two.example.com/login" {
		t.Fatalf("unexpected values: %#v", values)
	}
}
