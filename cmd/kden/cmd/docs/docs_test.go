package docs_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

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

		It("omits frontmatter by default", func() {
			dir := GinkgoT().TempDir()
			Expect(run("docs", "--type", "markdown", "--dir", dir)).To(Succeed())
			content, err := os.ReadFile(filepath.Join(dir, "cli.md"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).NotTo(HavePrefix("---\n"))
			Expect(string(content)).NotTo(ContainSubstring("\n---\n"))
		})

		It("prepends VitePress frontmatter with --frontmatter", func() {
			dir := GinkgoT().TempDir()
			Expect(run("docs", "--type", "markdown", "--dir", dir, "--frontmatter")).To(Succeed())
			content, err := os.ReadFile(filepath.Join(dir, "cli.md"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(HavePrefix("---\ntitle: CLI\n"))
			// Body after the frontmatter block starts at the command tree, no H1.
			body := string(content)
			if end := strings.Index(body[4:], "\n---\n"); end != -1 {
				body = body[4+end+len("\n---\n"):]
			}
			Expect(body).To(HavePrefix("\n## kden"))
			Expect(body).NotTo(ContainSubstring("\n---\n"))
		})
	})

	Context("unknown type", func() {
		It("returns an error", func() {
			Expect(run("docs", "--type", "html")).NotTo(Succeed())
		})
	})
})
