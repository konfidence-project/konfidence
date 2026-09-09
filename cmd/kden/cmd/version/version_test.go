package version_test

import (
	"bytes"
	"encoding/json"

	"github.com/konfidence-project/konfidence/cmd/kden/cmd/version"
	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
	"github.com/konfidence-project/konfidence/pkg/build"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// run executes the version command with the given --output value and returns
// stdout and stderr separately, so we can assert where the update hint lands.
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
			var info build.Info
			Expect(json.Unmarshal([]byte(stdout), &info)).To(Succeed())
			Expect(info.Platform).To(ContainSubstring("/")) // GOOS/GOARCH
			Expect(info.GoVersion).To(HavePrefix("go"))
		})

		It("emits yaml when requested", func() {
			stdout, _ := run("yaml")
			Expect(stdout).To(ContainSubstring("version:"))
			Expect(stdout).To(ContainSubstring("platform:"))
		})

		It("renders the pretty table without error", func() {
			// The pretty table is rendered by bubbletea directly to os.Stdout,
			// not the command buffer, so we can only assert it runs cleanly.
			Expect(func() { run("pretty") }).NotTo(Panic())
		})

		It("emits a bare version string for plain, nothing else", func() {
			orig := build.Version
			build.Version = "v1.2.3"
			defer func() { build.Version = orig }()

			stdout, stderr := run("plain")
			Expect(stdout).To(Equal("v1.2.3\n"))
			Expect(stdout).NotTo(ContainSubstring("commit"))
			Expect(stderr).To(BeEmpty()) // no update hint in plain
		})
	})

	Context("update hint", func() {
		It("always prints the hint to stderr for json, keeping stdout pipeable", func() {
			stdout, stderr := run("json")
			Expect(stdout).NotTo(ContainSubstring("install.sh"))
			Expect(stderr).To(ContainSubstring("install.sh | sh"))
		})

		It("always prints the hint to stderr for yaml", func() {
			stdout, stderr := run("yaml")
			Expect(stdout).NotTo(ContainSubstring("install.sh"))
			Expect(stderr).To(ContainSubstring("install.sh | sh"))
		})
	})
})
