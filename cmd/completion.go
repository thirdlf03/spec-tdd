package cmd

import (
	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate completion script",
	Long: `To load completions:

Bash:

  $ source <(app completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ app completion bash > /etc/bash_completion.d/app
  # macOS:
  $ app completion bash > $(brew --prefix)/etc/bash_completion.d/app

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ app completion zsh > "${fpath[1]}/_app"

  # You will need to start a new shell for this setup to take effect.

Fish:

  $ app completion fish | source

  # To load completions for each session, execute once:
  $ app completion fish > ~/.config/fish/completions/app.fish

PowerShell:

  PS> app completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> app completion powershell > app.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		out := cmd.OutOrStdout()
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletion(out)
		case "zsh":
			return cmd.Root().GenZshCompletion(out)
		case "fish":
			return cmd.Root().GenFishCompletion(out, true)
		case "powershell":
			return cmd.Root().GenPowerShellCompletionWithDesc(out)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
