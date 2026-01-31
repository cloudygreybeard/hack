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
	"sort"
	"strings"

	"github.com/cloudygreybeard/hack/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Version information set via ldflags at build time.
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "hack [filter]",
	Short: "Manage hack workspaces with pattern-based scaffolding",
	Long: `hack is a CLI tool for managing hack workspaces.

When called without arguments, it outputs the path to the most recently
modified hack directory. When called with a filter string, it finds and
outputs the first matching directory path.

Use with shell integration (run 'hack bootstrap') to enable automatic
directory changing.

Configuration is read from ~/.hack.yaml, environment variables (HACK_*),
and command-line flags.`,
	Args:               cobra.MaximumNArgs(1),
	DisableFlagParsing: false,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return config.Init()
	},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			dir, err := getMostRecentDir(config.C.RootDir)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			fmt.Println(dir)
			return
		}

		filter := args[0]
		dir, err := findMatchingDir(config.C.RootDir, filter)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if dir == "" {
			fmt.Fprintf(os.Stderr, "no directory matching %q found\n", filter)
			os.Exit(1)
		}
		fmt.Println(dir)
	},
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Persistent flags available to all commands
	rootCmd.PersistentFlags().String("root-dir", "", "root directory for hack workspaces")
	rootCmd.PersistentFlags().String("patterns-dir", "", "directory for patterns")
	rootCmd.PersistentFlags().String("config", "", "config file (default ~/.hack.yaml)")
	rootCmd.PersistentFlags().BoolP("interactive", "i", false, "enable interactive mode")

	// Bind flags to viper
	_ = viper.BindPFlag("root_dir", rootCmd.PersistentFlags().Lookup("root-dir"))
	_ = viper.BindPFlag("patterns_dir", rootCmd.PersistentFlags().Lookup("patterns-dir"))
	_ = viper.BindPFlag("interactive", rootCmd.PersistentFlags().Lookup("interactive"))
}

// getMostRecentDir returns the most recently modified directory in rootDir.
func getMostRecentDir(rootDir string) (string, error) {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		if os.IsNotExist(err) {
			return rootDir, nil
		}
		return "", fmt.Errorf("cannot read directory %s: %w", rootDir, err)
	}

	type dirWithTime struct {
		path    string
		modTime int64
	}

	var dirs []dirWithTime
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		dirs = append(dirs, dirWithTime{
			path:    filepath.Join(rootDir, entry.Name()),
			modTime: info.ModTime().Unix(),
		})
	}

	if len(dirs) == 0 {
		return rootDir, nil
	}

	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i].modTime > dirs[j].modTime
	})

	return dirs[0].path, nil
}

// findMatchingDir finds the first directory matching the filter (case-insensitive).
func findMatchingDir(rootDir, filter string) (string, error) {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return "", fmt.Errorf("cannot read directory %s: %w", rootDir, err)
	}

	filterLower := strings.ToLower(filter)
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if strings.Contains(strings.ToLower(entry.Name()), filterLower) {
			return filepath.Join(rootDir, entry.Name()), nil
		}
	}

	return "", nil
}
