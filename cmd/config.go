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
	"path/filepath"

	"github.com/cloudygreybeard/hack/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long: `View and manage configuration.

Configuration is loaded from (in order of precedence):
  1. Command-line flags
  2. Environment variables
  3. Config file
  4. Built-in defaults`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Run: func(cmd *cobra.Command, args []string) {
		if config.C.Persona != "" {
			fmt.Printf("# persona:     %s\n", config.C.Persona)
		}
		cfgFile := config.ConfigFilePath()
		if cfgFile != "" {
			fmt.Printf("# config file: %s\n", cfgFile)
		} else {
			fmt.Printf("# config file: not found, using defaults\n")
		}
		fmt.Printf("# env prefix:  %s_*\n", config.EnvPrefix())
		fmt.Println()

		type entry struct {
			key   string
			value string
		}

		entries := []entry{
			{"root_dir", config.C.RootDir},
			{"patterns_dir", config.C.PatternsDir},
			{"plugins_dir", config.C.PluginsDir},
			{"editor", config.C.Editor},
			{"ide", config.C.IDE},
			{"edit_mode", config.C.EditMode},
			{"git_init", fmt.Sprintf("%t", config.C.GitInit)},
			{"create_readme", fmt.Sprintf("%t", config.C.CreateReadme)},
			{"interactive", fmt.Sprintf("%t", config.C.Interactive)},
			{"default_org", config.C.DefaultOrg},
			{"shell_alias", config.C.ShellAlias},
		}

		maxKey := 0
		for _, e := range entries {
			if len(e.key) > maxKey {
				maxKey = len(e.key)
			}
		}

		for _, e := range entries {
			src := config.Source(e.key)
			val := e.value
			if val == "" {
				val = "(not set)"
			}
			fmt.Printf("%-*s  %-30s  # %s\n", maxKey+1, e.key+":", val, src)
		}
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a default config file",
	Long: `Create a default configuration file.

If the file already exists, this command will not overwrite it
unless --force is specified.`,
	Run: func(cmd *cobra.Command, args []string) {
		force, _ := cmd.Flags().GetBool("force")

		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error getting home directory: %v\n", err)
			os.Exit(1)
		}

		name := config.C.BinaryName()
		cfgPath := filepath.Join(home, "."+name+".yaml")

		if _, err := os.Stat(cfgPath); err == nil && !force {
			fmt.Fprintf(os.Stderr, "config file already exists: %s\n", cfgPath)
			fmt.Fprintln(os.Stderr, "use --force to overwrite")
			os.Exit(1)
		}

		dotBase := filepath.Join(home, ".hack")
		rootDir := filepath.Join(home, "hack")
		if config.C.Persona != "" {
			dotBase = filepath.Join(home, ".hack-"+config.C.Persona)
			rootDir = filepath.Join(home, "hack-"+config.C.Persona)
		}

		content := fmt.Sprintf(`# %s configuration
# See: %s config show

# Root directory for workspaces
root_dir: %s

# Directory for patterns (templates)
patterns_dir: %s/patterns

# Directory for plugins
plugins_dir: %s/plugins

# Terminal editor for %s edit --terminal and %s create
# editor: vim

# IDE command for %s edit --ide (e.g. cursor, code)
# ide: ""

# Edit mode: auto, terminal, or ide
# auto: use IDE if configured, otherwise terminal editor
edit_mode: auto

# Initialize git repository on create
git_init: true

# Create README.md on create
create_readme: true

# Enable interactive mode by default (prompt for pattern variables)
interactive: false

# Default GitHub organization for module paths
# If not set, uses example.com/<name> as the default module path
# default_org: your-org

# Short alias for shell integration (used by %s bootstrap)
# shell_alias: %s
`, name, name, rootDir, dotBase, dotBase, name, name, name, name, shortAliasSuggestion(config.C.Persona))

		if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing config file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Config file created: %s\n", cfgPath)
	},
}

func shortAliasSuggestion(persona string) string {
	if persona == "" {
		return "h"
	}
	return "h" + persona[:1]
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show config file path",
	Run: func(cmd *cobra.Command, args []string) {
		cfgFile := config.ConfigFilePath()
		if cfgFile != "" {
			fmt.Println(cfgFile)
		} else {
			home, _ := os.UserHomeDir()
			fmt.Println(filepath.Join(home, "."+config.C.BinaryName()+".yaml"))
		}
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configPathCmd)

	configInitCmd.Flags().Bool("force", false, "overwrite existing config file")
}
