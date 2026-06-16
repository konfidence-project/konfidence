package version_test

import (
	"bytes"

	"github.com/konfidence-project/konfidence/cmd/kden/cmd"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("versionCmd", func() {

	Context("called for version command", func() {
		It("should display version", func() {
			rootCmd := cmd.GetRootCommand()
			output := new(bytes.Buffer)
			rootCmd.SetOut(output)
			rootCmd.SetErr(output)
			rootCmd.SetArgs([]string{"version"})

			err := rootCmd.Execute()
			Expect(err).NotTo(HaveOccurred())
		})
	})

})
