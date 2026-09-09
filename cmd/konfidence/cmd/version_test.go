package cmd

import (
	"bytes"
	"encoding/json"

	"github.com/konfidence-project/konfidence/pkg/build"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("version command", func() {
	It("prints build metadata as JSON", func() {
		out := new(bytes.Buffer)
		rootCmd.SetOut(out)
		rootCmd.SetArgs([]string{"version", "--output", "json"})
		Expect(rootCmd.Execute()).To(Succeed())

		var info build.Info
		Expect(json.Unmarshal(out.Bytes(), &info)).To(Succeed())
		Expect(info.Version).NotTo(BeEmpty())
		Expect(info.Platform).To(ContainSubstring("/")) // GOOS/GOARCH
		Expect(info.GoVersion).To(HavePrefix("go"))
	})

	It("prints a bare version string with --output plain", func() {
		out := new(bytes.Buffer)
		rootCmd.SetOut(out)
		rootCmd.SetArgs([]string{"version", "--output", "plain"})
		Expect(rootCmd.Execute()).To(Succeed())

		Expect(out.String()).To(Equal(build.Current().Version + "\n"))
		Expect(out.String()).NotTo(ContainSubstring("{")) // not JSON
	})
})
