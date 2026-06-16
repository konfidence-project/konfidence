package completion

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const (
	bashCompletionArg       = "bash"
	zshCompletionArg        = "zsh"
	fishCompletionArg       = "fish"
	powershellCompletionArg = "powershell"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for kden.

To load completions:

Bash:
  $ source <(kden completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ kden completion bash > /etc/bash_completion.d/kden
  # macOS:
  $ kden completion bash > $(brew --prefix)/etc/bash_completion.d/kden

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. Execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ kden completion zsh > "${fpath[1]}/_kden"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ kden completion fish | source

  # To load completions for each session, execute once:
  $ kden completion fish > ~/.config/fish/completions/kden.fish

PowerShell:
  PS> kden completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> kden completion powershell > kden.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{bashCompletionArg, zshCompletionArg, fishCompletionArg, powershellCompletionArg},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case bashCompletionArg:
			err := cmd.Root().GenBashCompletion(os.Stdout)
			if err != nil {
				return fmt.Errorf("failed to generate bash completion: %w", err)
			}
		case zshCompletionArg:
			err := cmd.Root().GenZshCompletion(os.Stdout)
			if err != nil {
				return fmt.Errorf("failed to generate zsh completion: %w", err)
			}
		case fishCompletionArg:
			err := cmd.Root().GenFishCompletion(os.Stdout, true)
			if err != nil {
				return fmt.Errorf("failed to generate fish completion: %w", err)
			}
		case powershellCompletionArg:
			err := cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			if err != nil {
				return fmt.Errorf("failed to generate powershell completion: %w", err)
			}
		}
		return nil
	},
}

func NewCompletionCmd() *cobra.Command {
	return completionCmd
}
