package config_test

import (
	"bytes"

	kdencmd "github.com/konfidence-project/konfidence/cmd/kden/cmd"
	"github.com/konfidence-project/konfidence/cmd/kden/cmd/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var _ = Describe("config", func() {

	var rootCmd *cobra.Command

	BeforeEach(func() {
		rootCmd = kdencmd.GetRootCommand()
		rootCmd.AddCommand(config.NewConfigCmd())
	})

	Describe("set", func() {
		Context("with valid input", func() {
			It("should set the configuration value", func() {
				output := new(bytes.Buffer)
				rootCmd.SetOut(output)
				rootCmd.SetErr(output)
				rootCmd.SetArgs([]string{"config", "set", "log-level", "info"})

				err := rootCmd.Execute()

				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("with invalid input", func() {
			It("should throw an error when there are less than 2 arguments", func() {
				output := new(bytes.Buffer)
				rootCmd.SetOut(output)
				rootCmd.SetErr(output)
				rootCmd.SetArgs([]string{"config", "set", "log-level"})

				err := rootCmd.Execute()

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("accepts 2 arg(s), received 1"))
			})

			It("should throw an error when there are more than 2 arguments", func() {
				output := new(bytes.Buffer)
				rootCmd.SetOut(output)
				rootCmd.SetErr(output)
				rootCmd.SetArgs([]string{"config", "set", "log-level", "info", "extra-arg"})

				err := rootCmd.Execute()

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("accepts 2 arg(s), received 3"))
			})

			It("should throw an error when there is error during config setting", func() {
				output := new(bytes.Buffer)
				rootCmd.SetOut(output)
				rootCmd.SetErr(output)
				rootCmd.SetArgs([]string{"config", "set", "log-level", "invalid"})

				err := rootCmd.Execute()

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to set configuration"))
				Expect(err.Error()).To(ContainSubstring("value 'invalid' is not valid for configuration key 'log-level'"))
			})
		})
	})

	Describe("unset", func() {
		Context("with valid input", func() {
			It("should unset the configuration value", func() {
				output := new(bytes.Buffer)
				rootCmd.SetOut(output)
				rootCmd.SetErr(output)
				rootCmd.SetArgs([]string{"config", "unset", "log-level"})

				err := rootCmd.Execute()

				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("with invalid input", func() {
			It("should throw an error when there are no arguments", func() {
				output := new(bytes.Buffer)
				rootCmd.SetOut(output)
				rootCmd.SetErr(output)
				rootCmd.SetArgs([]string{"config", "unset"})

				err := rootCmd.Execute()

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("accepts 1 arg(s), received 0"))
			})

			It("should throw an error when there is more than 1 argument", func() {
				output := new(bytes.Buffer)
				rootCmd.SetOut(output)
				rootCmd.SetErr(output)
				rootCmd.SetArgs([]string{"config", "unset", "log-level", "info"})

				err := rootCmd.Execute()

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("accepts 1 arg(s), received 2"))
			})

			It("should throw an error when there is error during config unsetting", func() {
				output := new(bytes.Buffer)
				rootCmd.SetOut(output)
				rootCmd.SetErr(output)
				rootCmd.SetArgs([]string{"config", "unset", "invalid-config"})

				err := rootCmd.Execute()

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to unset configuration \"invalid-config\""))
				Expect(err.Error()).To(ContainSubstring("'invalid-config' is not a valid configuration key"))
			})
		})
	})
})
