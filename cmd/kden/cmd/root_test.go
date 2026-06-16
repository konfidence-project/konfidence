package cmd

import (
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
