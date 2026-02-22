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
	Use:               "show <name>",
	Short:             "Show details about a pattern",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completePatterns,
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
		if p.Weight != 0 {
			fmt.Printf("Weight:      %d\n", p.Weight)
		}

		if len(p.Labels) > 0 {
			fmt.Println("\nLabels:")
			for k, v := range p.Labels {
				fmt.Printf("  %s=%s\n", k, v)
			}
		}

		if len(p.DefaultLabels) > 0 {
			fmt.Println("\nDefault workspace labels:")
			for k, v := range p.DefaultLabels {
				fmt.Printf("  %s=%s\n", k, v)
			}
		}

		if len(p.Inherits) > 0 {
			fmt.Println("\nInherits:")
			for _, inh := range p.Inherits {
				if inh.Pattern != "" {
					fmt.Printf("  - pattern: %s\n", inh.Pattern)
				}
				if inh.PatternSelector != nil {
					fmt.Printf("  - patternSelector:\n")
					for k, v := range inh.PatternSelector.MatchLabels {
						fmt.Printf("      %s=%s\n", k, v)
					}
				}
			}
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

		if len(p.PostCreate) > 0 {
			fmt.Println("\nPost-create hooks:")
			for i, hook := range p.PostCreate {
				fmt.Printf("  %d. %s\n", i+1, hook)
			}
		}
	},
}

var patternSyncCmd = &cobra.Command{
	Use:   "sync <directory>",
	Short: "Bulk install patterns from a directory",
	Long: `Install all patterns found in a directory.

Each subdirectory containing a pattern.yaml is treated as a pattern
and installed to ~/.hack/patterns/.

This is useful for syncing patterns from a development workspace
or a shared patterns repository.

Examples:
  hack pattern sync ./patterns
  hack pattern sync ~/my-patterns`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		srcDir := args[0]

		entries, err := os.ReadDir(srcDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading directory: %v\n", err)
			os.Exit(1)
		}

		var installed, skipped int
		for _, entry := range entries {
			if !entry.IsDir() || entry.Name()[0] == '.' {
				continue
			}
			patternDir := fmt.Sprintf("%s/%s", srcDir, entry.Name())

			// Check for pattern.yaml
			if _, err := os.Stat(patternDir + "/pattern.yaml"); os.IsNotExist(err) {
				skipped++
				continue
			}

			if err := pattern.Install(patternDir, config.C.PatternsDir); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to install %s: %v\n", entry.Name(), err)
				skipped++
				continue
			}

			fmt.Printf("installed: %s\n", entry.Name())
			installed++
		}

		fmt.Printf("\n%d pattern(s) installed, %d skipped\n", installed, skipped)
	},
}

func init() {
	rootCmd.AddCommand(patternCmd)
	patternCmd.AddCommand(patternListCmd)
	patternCmd.AddCommand(patternInstallCmd)
	patternCmd.AddCommand(patternShowCmd)
	patternCmd.AddCommand(patternSyncCmd)
}
