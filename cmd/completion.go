// Copyright 2026 cloudygreybeard
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate completion scripts for your shell.

To load completions:

Bash:
  $ source <(hack completion bash)

  To load completions for each session, execute once:
  Linux:
    $ hack completion bash > /etc/bash_completion.d/hack
  macOS:
    $ hack completion bash > $(brew --prefix)/etc/bash_completion.d/hack

Zsh:
  $ source <(hack completion zsh)

  To load completions for each session, execute once:
    $ hack completion zsh > "${fpath[1]}/_hack"

Fish:
  $ hack completion fish | source

  To load completions for each session, execute once:
    $ hack completion fish > ~/.config/fish/completions/hack.fish

PowerShell:
  PS> hack completion powershell | Out-String | Invoke-Expression

  To load completions for every new session, add the output to your profile.`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			_ = rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			_ = rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			_ = rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			_ = rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
