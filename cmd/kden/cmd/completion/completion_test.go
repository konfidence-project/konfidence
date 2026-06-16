package completion_test

import (
	"bytes"

	kdencmd "github.com/konfidence-project/konfidence/cmd/kden/cmd"
	"github.com/konfidence-project/konfidence/cmd/kden/cmd/completion"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var _ = Describe("completionCmd", func() {

	var rootCmd *cobra.Command

	BeforeEach(func() {
		rootCmd = kdencmd.GetRootCommand()
		rootCmd.AddCommand(completion.NewCompletionCmd())
	})

	DescribeTable("generate completion for valid shells",
		func(shell string) {
			output := new(bytes.Buffer)
			rootCmd.SetOut(output)
			rootCmd.SetErr(output)

			rootCmd.SetArgs([]string{"completion", shell})
			err := rootCmd.Execute()

			Expect(err).NotTo(HaveOccurred())
		},
		Entry("use autocompletion in bash", "bash"),
		Entry("use autocompletion in zsh", "zsh"),
		Entry("use autocompletion in fish", "fish"),
		Entry("use autocompletion in powershell", "powershell"),
	)

	Context("when called with invalid args", func() {

		It("should return an error for an unknown shell", func() {
			output := new(bytes.Buffer)
			rootCmd.SetOut(output)
			rootCmd.SetErr(output)
			rootCmd.SetArgs([]string{"completion", "invalidshell"})
			err := rootCmd.Execute()
			Expect(err).To(HaveOccurred())
		})

		It("should return an error when no shell is provided", func() {
			output := new(bytes.Buffer)
			rootCmd.SetOut(output)
			rootCmd.SetErr(output)
			rootCmd.SetArgs([]string{"completion"})
			err := rootCmd.Execute()
			Expect(err).To(HaveOccurred())
		})

		It("should return an error when too many args are provided", func() {
			output := new(bytes.Buffer)
			rootCmd.SetOut(output)
			rootCmd.SetErr(output)
			rootCmd.SetArgs([]string{"completion", "bash", "zsh"})
			err := rootCmd.Execute()
			Expect(err).To(HaveOccurred())
		})
	})
})
