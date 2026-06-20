package main

import (
	"fmt"

	sdk "github.com/inovacc/selo"
	"github.com/spf13/cobra"
)

// newCompletionCmd builds the "completion" command that emits a shell-completion
// script for the user's shell. Cobra's auto-generated completion command is
// disabled on the root (see newRootCmd) in favour of this explicit one, which
// carries per-shell install instructions. The hidden "__complete" engine that
// the generated scripts invoke at runtime is registered by Cobra during
// Execute regardless of DisableDefaultCmd, so dynamic completion still works.
func newCompletionCmd() *cobra.Command {
	bin := sdk.CLIUse

	long := fmt.Sprintf(`Generate a shell completion script for %s.

Load completions for the current session or persist them:

Bash:
  $ source <(%[2]s completion bash)
  # Linux, persisted:
  $ %[2]s completion bash > /etc/bash_completion.d/%[2]s

Zsh:
  $ %[2]s completion zsh > "${fpath[1]}/_%[2]s"

Fish:
  $ %[2]s completion fish | source
  $ %[2]s completion fish > ~/.config/fish/completions/%[2]s.fish

PowerShell:
  PS> %[2]s completion powershell | Out-String | Invoke-Expression
`, sdk.AppName, bin)

	return &cobra.Command{
		Use:                   "completion [bash|zsh|fish|powershell]",
		Short:                 "Generate a shell completion script for " + sdk.AppName,
		Long:                  long,
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()

			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletionV2(w, true)
			case "zsh":
				return cmd.Root().GenZshCompletion(w)
			case "fish":
				return cmd.Root().GenFishCompletion(w, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(w)
			default:
				return fmt.Errorf("unsupported shell %q", args[0])
			}
		},
	}
}
