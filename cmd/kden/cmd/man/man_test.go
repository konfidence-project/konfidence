package man_test

import (
	"bytes"
	"os"
	"path/filepath"

	kdencmd "github.com/konfidence-project/konfidence/cmd/kden/cmd"
	"github.com/konfidence-project/konfidence/cmd/kden/cmd/man"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var _ = Describe("docsCmd/manCmd", func() {

	type tmpDirPath struct {
		setPath func(string) error
	}

	var tmpDirPathFuncs tmpDirPath
	var manDirPath string
	var rootCmd *cobra.Command

	BeforeEach(func() {
		rootCmd = kdencmd.GetRootCommand()
		rootCmd.AddCommand(man.NewManCmd())
		tempDir := GinkgoT().TempDir()
		manDirPath = tempDir

		tmpDirPathFuncs = tmpDirPath{
			setPath: func(path string) error {
				if path == "" {
					manDirPath = tempDir
					return nil
				}
				manDirPath = filepath.Join(tempDir, path)
				return os.MkdirAll(filepath.Dir(manDirPath), os.ModePerm)
			},
		}
	})

	Context("create docs man pages", func() {

		It("should generate man for tmp directory", func() {
			errCreate := tmpDirPathFuncs.setPath("tmp")
			Expect(errCreate).NotTo(HaveOccurred())

			output := new(bytes.Buffer)
			rootCmd.SetOut(output)
			rootCmd.SetErr(output)
			rootCmd.SetArgs([]string{"docs", "man", manDirPath})

			err := rootCmd.Execute()
			Expect(err).NotTo(HaveOccurred())
			Expect(manDirPath).ToNot(BeEmpty())
		})

		It("should generate man for default ./man directory", func() {
			output := new(bytes.Buffer)
			rootCmd.SetOut(output)
			rootCmd.SetErr(output)
			rootCmd.SetArgs([]string{"docs", "man", manDirPath})

			err := rootCmd.Execute()
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
