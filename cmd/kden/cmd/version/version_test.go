package version_test

import (
	"bytes"

	"github.com/konfidence-project/konfidence/cmd/kden/cmd/version"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("versionCmd", func() {

	Context("called for version command", func() {
		It("should display version", func() {
			versionCmd := version.NewVersionCmd()
			output := new(bytes.Buffer)
			versionCmd.SetOut(output)
			versionCmd.SetErr(output)

			err := versionCmd.Execute()
			Expect(err).NotTo(HaveOccurred())
			Expect(output.String()).To(ContainSubstring("kden CLI version:"))
		})
	})

})
