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
	"fmt"
	"os"

	"github.com/cloudygreybeard/hack/internal/config"
	"github.com/cloudygreybeard/hack/internal/pattern"
	"github.com/spf13/cobra"
)

var patternCmd = &cobra.Command{
	Use:   "pattern",
	Short: "Manage project patterns",
	Long: `Manage project patterns for scaffolding new workspaces.

Patterns are stored in ~/.hack/patterns/ and can be applied
when creating new workspaces with 'hack create -p <pattern>'.`,
}

var patternListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List available patterns",
	Run: func(cmd *cobra.Command, args []string) {
		patterns, err := pattern.List(config.C.PatternsDir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Fprintln(os.Stderr, "no patterns installed")
				fmt.Fprintf(os.Stderr, "use 'hack pattern install <path>' to install patterns\n")
				return
			}
			fmt.Fprintf(os.Stderr, "error listing patterns: %v\n", err)
			os.Exit(1)
		}

		if len(patterns) == 0 {
			fmt.Fprintln(os.Stderr, "no patterns installed")
			fmt.Fprintf(os.Stderr, "use 'hack pattern install <path>' to install patterns\n")
			return
		}

		for _, p := range patterns {
			if p.Description != "" {
				fmt.Printf("%-15s %s\n", p.Name, p.Description)
			} else {
				fmt.Println(p.Name)
			}
		}
	},
}

var patternInstallCmd = &cobra.Command{
	Use:   "install <path>",
	Short: "Install a pattern from a directory",
	Long: `Install a pattern from a local directory to ~/.hack/patterns/.

The source directory should contain:
  - pattern.yaml     Pattern metadata
  - template/        Files to copy when applying the pattern

Examples:
  hack pattern install ./patterns/go-cli
  hack pattern install ~/my-patterns/web-app`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		srcPath := args[0]

		if err := pattern.Install(srcPath, config.C.PatternsDir); err != nil {
			fmt.Fprintf(os.Stderr, "error installing pattern: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("pattern installed to %s\n", config.C.PatternsDir)
	},
}

var patternShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show details about a pattern",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		patternPath := fmt.Sprintf("%s/%s", config.C.PatternsDir, name)

		p, err := pattern.Load(patternPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading pattern: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Name:        %s\n", p.Name)
		if p.Description != "" {
			fmt.Printf("Description: %s\n", p.Description)
		}

		if len(p.Variables) > 0 {
			fmt.Println("\nVariables:")
			for _, v := range p.Variables {
				req := ""
				if v.Required {
					req = " (required)"
				}
				def := ""
				if v.Default != "" {
					def = fmt.Sprintf(" [default: %s]", v.Default)
				}
				fmt.Printf("  %-12s %s%s%s\n", v.Name, v.Description, req, def)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(patternCmd)
	patternCmd.AddCommand(patternListCmd)
	patternCmd.AddCommand(patternInstallCmd)
	patternCmd.AddCommand(patternShowCmd)
}
