package cmd

import (
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var _ = Describe("rootCmd", func() {

	Context("pass correct flags to root command", func() {

		It("should display help flag correctly", func() {
			output := new(bytes.Buffer)
			rootCmd.SetOut(output)
			rootCmd.SetErr(output)
			rootCmd.SetArgs([]string{"--help"})

			err := rootCmd.Execute()

			Expect(err).ToNot(HaveOccurred())
			Expect(output.String()).To(ContainSubstring("kden help"))
		})
	})

	Context("pass incorrect flags to root command", func() {

		It("should error on unknown flag", func() {
			rootCmd.SetArgs([]string{"--exists"})

			err := rootCmd.Execute()

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("unknown flag: --exists"))
		})
	})
})

var _ = Describe("resolveAccessToken", func() {
	newCommand := func() *cobra.Command {
		cmd := &cobra.Command{Use: "test"}
		cmd.Flags().String("access-token", "", "")
		return cmd
	}

	It("returns no token when neither source is set", func() {
		GinkgoT().Setenv("KDEN_ACCESS_TOKEN", "")
		token, err := resolveAccessToken(newCommand())
		Expect(err).NotTo(HaveOccurred())
		Expect(token).To(BeEmpty())
	})

	It("reads the token from the environment", func() {
		GinkgoT().Setenv("KDEN_ACCESS_TOKEN", "environment-token")
		token, err := resolveAccessToken(newCommand())
		Expect(err).NotTo(HaveOccurred())
		Expect(token).To(Equal("environment-token"))
	})

	It("prefers the flag over the environment", func() {
		GinkgoT().Setenv("KDEN_ACCESS_TOKEN", "environment-token")
		cmd := newCommand()
		Expect(cmd.Flags().Set("access-token", "flag-token")).To(Succeed())

		token, err := resolveAccessToken(cmd)
		Expect(err).NotTo(HaveOccurred())
		Expect(token).To(Equal("flag-token"))
	})

	It("rejects only whitespace or empty flag", func() {
		cmd := newCommand()
		Expect(cmd.Flags().Set("access-token", "    ")).To(Succeed())

		_, err := resolveAccessToken(cmd)
		Expect(err).To(MatchError(
			"access token is empty",
		))
	})
	It("trims flag", func() {
		cmd := newCommand()
		Expect(cmd.Flags().Set("access-token", " test    ")).To(Succeed())

		token, err := resolveAccessToken(cmd)
		Expect(err).NotTo(HaveOccurred())
		Expect(token).To(Equal("test"))
	})
	It("trims env var", func() {
		GinkgoT().Setenv("KDEN_ACCESS_TOKEN", "   env-test  ")
		cmd := newCommand()
		token, err := resolveAccessToken(cmd)
		Expect(err).NotTo(HaveOccurred())
		Expect(token).To(Equal("env-test"))
	})
})
