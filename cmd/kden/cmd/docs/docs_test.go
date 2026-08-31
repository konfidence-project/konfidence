package docs_test

import (
	"bytes"
	"os"
	"path/filepath"

	kdencmd "github.com/konfidence-project/konfidence/cmd/kden/cmd"
	"github.com/konfidence-project/konfidence/cmd/kden/cmd/docs"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var _ = Describe("docsCmd", func() {
	var rootCmd *cobra.Command

	BeforeEach(func() {
		rootCmd = kdencmd.GetRootCommand()
		rootCmd.AddCommand(docs.NewDocsCmd())
	})

	run := func(args ...string) error {
		out := new(bytes.Buffer)
		rootCmd.SetOut(out)
		rootCmd.SetErr(out)
		rootCmd.SetArgs(args)
		return rootCmd.Execute()
	}

	Context("--type man", func() {
		It("generates man pages into the given directory", func() {
			dir := GinkgoT().TempDir()
			Expect(run("docs", "--type", "man", "--dir", dir)).To(Succeed())
			entries, err := os.ReadDir(dir)
			Expect(err).NotTo(HaveOccurred())
			Expect(entries).NotTo(BeEmpty())
			Expect(entries[0].Name()).To(HaveSuffix(".1"))
		})
	})

	Context("--type markdown", func() {
		It("generates a single merged cli.md", func() {
			dir := GinkgoT().TempDir()
			Expect(run("docs", "--type", "markdown", "--dir", dir)).To(Succeed())
			content, err := os.ReadFile(filepath.Join(dir, "cli.md"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("## kden"))
		})

		It("does not contain --- HR separators", func() {
			dir := GinkgoT().TempDir()
			Expect(run("docs", "--type", "markdown", "--dir", dir)).To(Succeed())
			content, err := os.ReadFile(filepath.Join(dir, "cli.md"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).NotTo(ContainSubstring("\n---\n"))
		})
	})

	Context("unknown type", func() {
		It("returns an error", func() {
			Expect(run("docs", "--type", "html")).NotTo(Succeed())
		})
	})
})
