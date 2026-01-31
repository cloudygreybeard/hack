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
	Long: `View and manage hack configuration.

Configuration is loaded from (in order of precedence):
  1. Command-line flags
  2. Environment variables (HACK_*)
  3. Config file (~/.hack.yaml)
  4. Built-in defaults`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Run: func(cmd *cobra.Command, args []string) {
		cfgFile := config.ConfigFilePath()
		if cfgFile != "" {
			fmt.Printf("Config file: %s\n\n", cfgFile)
		} else {
			fmt.Println("Config file: (not found, using defaults)")
			fmt.Println()
		}

		fmt.Printf("root_dir:      %s\n", config.C.RootDir)
		fmt.Printf("patterns_dir:  %s\n", config.C.PatternsDir)
		fmt.Printf("editor:        %s\n", config.C.Editor)
		fmt.Printf("git_init:      %t\n", config.C.GitInit)
		fmt.Printf("create_readme: %t\n", config.C.CreateReadme)
		fmt.Printf("interactive:   %t\n", config.C.Interactive)
		if config.C.DefaultOrg != "" {
			fmt.Printf("default_org:   %s\n", config.C.DefaultOrg)
		} else {
			fmt.Printf("default_org:   (not set, using example.com)\n")
		}
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a default config file",
	Long: `Create a default configuration file at ~/.hack.yaml.

If the file already exists, this command will not overwrite it
unless --force is specified.`,
	Run: func(cmd *cobra.Command, args []string) {
		force, _ := cmd.Flags().GetBool("force")

		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error getting home directory: %v\n", err)
			os.Exit(1)
		}

		cfgPath := filepath.Join(home, ".hack.yaml")

		// Check if file exists
		if _, err := os.Stat(cfgPath); err == nil && !force {
			fmt.Fprintf(os.Stderr, "config file already exists: %s\n", cfgPath)
			fmt.Fprintln(os.Stderr, "use --force to overwrite")
			os.Exit(1)
		}

		content := `# hack configuration
# See: hack config show

# Root directory for hack workspaces
root_dir: ~/hack

# Directory for patterns (templates)
patterns_dir: ~/.hack/patterns

# Editor to open README.md after creating workspace
# editor: vim

# Initialize git repository on create
git_init: true

# Create README.md on create
create_readme: true

# Enable interactive mode by default (prompt for pattern variables)
interactive: false

# Default GitHub organization for module paths
# If not set, uses example.com/<name> as the default module path
# default_org: your-org
`

		if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing config file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Config file created: %s\n", cfgPath)
	},
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
			fmt.Println(filepath.Join(home, ".hack.yaml"))
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
