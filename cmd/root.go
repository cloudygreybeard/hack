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
	"strconv"
	"strings"

	"github.com/cloudygreybeard/hack/internal/config"
	"github.com/cloudygreybeard/hack/internal/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var verbosity int
var quiet bool

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
	ValidArgsFunction:  completeWorkspaces,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Set log verbosity
		if quiet {
			log.SetLevel(log.LevelQuiet)
		} else {
			log.SetVerbosity(verbosity)
		}
		return config.Init()
	},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			dir, err := getMostRecentDir(config.C.RootDir)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			outputCdTarget(dir)
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
		outputCdTarget(dir)
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
	rootCmd.PersistentFlags().CountVarP(&verbosity, "verbose", "v", "increase verbosity (-v, -vv, -vvv)")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "suppress informational output")

	// Bind flags to viper
	_ = viper.BindPFlag("root_dir", rootCmd.PersistentFlags().Lookup("root-dir"))
	_ = viper.BindPFlag("patterns_dir", rootCmd.PersistentFlags().Lookup("patterns-dir"))
	_ = viper.BindPFlag("interactive", rootCmd.PersistentFlags().Lookup("interactive"))
}

// outputCdTarget writes a path to a file descriptor specified by HACK_CD_FD
// (for shell wrapper), otherwise falls back to stdout. This allows the shell
// wrapper to capture the cd target while leaving stdout free for interactive
// use (editor).
func outputCdTarget(path string) {
	if fdStr := os.Getenv("HACK_CD_FD"); fdStr != "" {
		if fdNum, err := strconv.Atoi(fdStr); err == nil && fdNum > 2 {
			fd := os.NewFile(uintptr(fdNum), "cd_target")
			if fd != nil {
				if _, err := fd.Stat(); err == nil {
					fmt.Fprintln(fd, path)
					fd.Close()
					return
				}
				fd.Close()
			}
		}
	}
	// Fall back to stdout
	fmt.Println(path)
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

// findMatchingDir finds the best-matching directory for the filter.
// Results are ranked: exact suffix match > word-boundary match > substring match.
// Within each tier, more recently modified directories rank higher.
func findMatchingDir(rootDir, filter string) (string, error) {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return "", fmt.Errorf("cannot read directory %s: %w", rootDir, err)
	}

	filterLower := strings.ToLower(filter)

	type scored struct {
		path    string
		score   int
		modTime int64
	}

	var matches []scored
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		nameLower := strings.ToLower(entry.Name())
		if !strings.Contains(nameLower, filterLower) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		score := scoreMatch(nameLower, filterLower)
		matches = append(matches, scored{
			path:    filepath.Join(rootDir, entry.Name()),
			score:   score,
			modTime: info.ModTime().Unix(),
		})
	}

	if len(matches) == 0 {
		return "", nil
	}

	// Sort by score (descending), then by modification time (most recent first)
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		return matches[i].modTime > matches[j].modTime
	})

	return matches[0].path, nil
}

// scoreMatch returns a ranking score for how well name matches filter.
// Higher scores indicate better matches.
func scoreMatch(name, filter string) int {
	// Exact match on the concept portion (after date prefix)
	parts := strings.SplitN(name, ".", 2)
	if len(parts) == 2 {
		concept := parts[1]
		if concept == filter {
			return 100
		}
		if strings.HasSuffix(concept, filter) {
			return 80
		}
		if strings.HasPrefix(concept, filter) {
			return 70
		}
	}

	// Exact full match
	if name == filter {
		return 90
	}

	// Word-boundary match (filter appears after a separator)
	for _, sep := range []string{".", "-", "_"} {
		if strings.Contains(name, sep+filter) {
			return 60
		}
	}

	// Suffix match
	if strings.HasSuffix(name, filter) {
		return 50
	}

	// Prefix match
	if strings.HasPrefix(name, filter) {
		return 40
	}

	// General substring match
	return 10
}

// completeWorkspaces provides shell completion for workspace names.
func completeWorkspaces(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	_ = config.Init()
	entries, err := os.ReadDir(config.C.RootDir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var names []string
	toCompleteLower := strings.ToLower(toComplete)
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if toComplete == "" || strings.Contains(strings.ToLower(entry.Name()), toCompleteLower) {
			names = append(names, entry.Name())
		}
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

// completePatterns provides shell completion for pattern names.
func completePatterns(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	_ = config.Init()
	entries, err := os.ReadDir(config.C.PatternsDir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var names []string
	toCompleteLower := strings.ToLower(toComplete)
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if toComplete == "" || strings.Contains(strings.ToLower(entry.Name()), toCompleteLower) {
			names = append(names, entry.Name())
		}
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}
