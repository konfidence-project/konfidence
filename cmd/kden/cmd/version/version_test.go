package version_test

import (
	"bytes"
	"encoding/json"

	"github.com/konfidence-project/konfidence/cmd/kden/cmd/version"
	cfg "github.com/konfidence-project/konfidence/internal/kden/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// run executes the version command with the given --output value and returns
// stdout and stderr separately, so we can assert the update hint never leaks
// into the machine-readable stdout.
func run(outputFormat string) (stdout, stderr string) {
	cfg.Config.Output = outputFormat
	cmd := version.NewVersionCmd()
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(errOut)
	cmd.SetArgs([]string{})
	Expect(cmd.Execute()).To(Succeed())
	return out.String(), errOut.String()
}

var _ = Describe("versionCmd", func() {
	Context("output formats", func() {
		It("emits valid JSON by default with the expected fields", func() {
			stdout, _ := run("json")
			var info version.Info
			Expect(json.Unmarshal([]byte(stdout), &info)).To(Succeed())
			Expect(info.Platform).To(ContainSubstring("/")) // GOOS/GOARCH
			Expect(info.GoVersion).To(HavePrefix("go"))
		})

		It("emits yaml when requested", func() {
			stdout, _ := run("yaml")
			Expect(stdout).To(ContainSubstring("version:"))
			Expect(stdout).To(ContainSubstring("platform:"))
		})

		It("emits an aligned human block for pretty", func() {
			stdout, _ := run("pretty")
			Expect(stdout).To(HavePrefix("kden "))
			Expect(stdout).To(ContainSubstring("Platform:"))
			Expect(stdout).To(ContainSubstring("Commit:"))
		})
	})

	Context("update hint", func() {
		It("keeps the hint out of stdout (dev build prints no hint at all)", func() {
			// The test binary reports build.Version == "dev": no hint anywhere.
			stdout, stderr := run("json")
			Expect(stdout).NotTo(ContainSubstring("re-run"))
			Expect(stderr).NotTo(ContainSubstring("re-run"))
		})
	})
})
