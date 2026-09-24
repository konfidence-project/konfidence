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
	t.Setenv("API_OIDC_ALLOW_RETURN_URLS", "one.example.com,two.example.com")

	values := envList("API_OIDC_ALLOW_RETURN_URLS")
	if len(values) != 2 || values[0] != "one.example.com" || values[1] != "two.example.com" {
		t.Fatalf("unexpected values: %#v", values)
	}
}

func TestJWKSCacheTTLFlagDefault(t *testing.T) {
	flag := rootCmd.Flags().Lookup("oidc-jwks-cache-ttl")
	if flag == nil {
		t.Fatal("oidc-jwks-cache-ttl flag is not registered")
	}
	if flag.DefValue != "15m" {
		t.Fatalf("expected default JWKS cache TTL 15m, got %q", flag.DefValue)
	}
}
