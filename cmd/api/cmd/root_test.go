package cmd

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

func setEnv(key, value string) {
	GinkgoHelper()

	original, wasSet := os.LookupEnv(key)
	Expect(os.Setenv(key, value)).To(Succeed())
	DeferCleanup(func() {
		if wasSet {
			Expect(os.Setenv(key, original)).To(Succeed())
			return
		}
		Expect(os.Unsetenv(key)).To(Succeed())
	})
}

var _ = Describe("root command", Serial, func() {
	BeforeEach(func() {
		originalCfg := cfg
		DeferCleanup(func() { cfg = originalCfg })
	})

	It("loads the OIDC client secret from the environment", func() {
		setEnv("API_OIDC_CLIENT_SECRET", "environment-secret")
		cfg.OIDC.ClientSecret = ""
		cmd := &cobra.Command{}
		cmd.Flags().String("oidc-client-secret", "", "")

		loadOIDCClientSecret(cmd, nil)

		Expect(cfg.OIDC.ClientSecret).To(Equal("environment-secret"))
	})

	It("preserves an explicit OIDC client secret flag value", func() {
		setEnv("API_OIDC_CLIENT_SECRET", "environment-secret")
		cfg.OIDC.ClientSecret = "flag-secret"
		cmd := &cobra.Command{}
		cmd.Flags().String("oidc-client-secret", "", "")
		Expect(cmd.Flags().Set("oidc-client-secret", "flag-secret")).To(Succeed())

		loadOIDCClientSecret(cmd, nil)

		Expect(cfg.OIDC.ClientSecret).To(Equal("flag-secret"))
	})

	It("reads a comma-separated environment list", func() {
		setEnv("API_OIDC_ALLOWED_RETURN_HOSTS", "one.example.com,two.example.com")

		values := envList("API_OIDC_ALLOWED_RETURN_HOSTS")

		Expect(values).To(Equal([]string{"one.example.com", "two.example.com"}))
	})

	It("registers the default JWKS cache TTL flag", func() {
		flag := rootCmd.Flags().Lookup("oidc-jwks-cache-ttl")
		Expect(flag).NotTo(BeNil())
		Expect(flag.DefValue).To(Equal("15m"))
	})
})
