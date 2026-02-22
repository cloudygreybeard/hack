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
	"github.com/cloudygreybeard/hack/internal/log"
	"github.com/spf13/cobra"
)

var archiveCmd = &cobra.Command{
	Use:     "archive <filter>",
	Aliases: []string{"ar"},
	Short:   "Archive a hack workspace",
	Long: `Archive a hack workspace by moving it to an .archive subdirectory.

Archived workspaces are hidden from listing and navigation but can
be restored by moving them back to the root directory.

The archive directory is created at <root_dir>/.archive/ if it
does not already exist.

Examples:
  hack archive old-project       # Archive matching workspace
  hack archive --list            # List archived workspaces
  hack archive --restore proj    # Restore an archived workspace`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeWorkspaces,
	Run: func(cmd *cobra.Command, args []string) {
		archiveDir := filepath.Join(config.C.RootDir, ".archive")

		if archiveList {
			listArchived(archiveDir)
			return
		}

		if archiveRestore != "" {
			restoreArchived(archiveDir, archiveRestore)
			return
		}

		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "workspace filter required (or use --list)")
			os.Exit(1)
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

		dirName := filepath.Base(dir)

		// Create archive directory
		if err := os.MkdirAll(archiveDir, 0755); err != nil {
			log.Error("creating archive directory: %v", err)
			os.Exit(1)
		}

		destPath := filepath.Join(archiveDir, dirName)
		if _, err := os.Stat(destPath); err == nil {
			log.Error("archive already contains %q; remove or rename it first", dirName)
			os.Exit(1)
		}

		log.Verbose("archiving workspace: %s", dirName)
		if err := os.Rename(dir, destPath); err != nil {
			log.Error("archiving workspace: %v", err)
			os.Exit(1)
		}

		fmt.Fprintf(os.Stderr, "archived: %s\n", dirName)
	},
}

var (
	archiveList    bool
	archiveRestore string
)

func init() {
	rootCmd.AddCommand(archiveCmd)

	archiveCmd.Flags().BoolVar(&archiveList, "list", false, "list archived workspaces")
	archiveCmd.Flags().StringVar(&archiveRestore, "restore", "", "restore an archived workspace")
}

func listArchived(archiveDir string) {
	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintln(os.Stderr, "no archived workspaces")
			return
		}
		fmt.Fprintf(os.Stderr, "error reading archive: %v\n", err)
		os.Exit(1)
	}

	if len(entries) == 0 {
		fmt.Fprintln(os.Stderr, "no archived workspaces")
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			fmt.Println(entry.Name())
		}
	}
}

func restoreArchived(archiveDir, filter string) {
	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintln(os.Stderr, "no archived workspaces")
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "error reading archive: %v\n", err)
		os.Exit(1)
	}

	filterLower := filter
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !containsIgnoreCase(name, filterLower) {
			continue
		}

		srcPath := filepath.Join(archiveDir, name)
		destPath := filepath.Join(config.C.RootDir, name)

		if _, err := os.Stat(destPath); err == nil {
			log.Error("workspace %q already exists in root directory", name)
			os.Exit(1)
		}

		log.Verbose("restoring workspace: %s", name)
		if err := os.Rename(srcPath, destPath); err != nil {
			log.Error("restoring workspace: %v", err)
			os.Exit(1)
		}

		fmt.Fprintf(os.Stderr, "restored: %s\n", name)
		return
	}

	fmt.Fprintf(os.Stderr, "no archived workspace matching %q found\n", filter)
	os.Exit(1)
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(substr) == 0 ||
			findIgnoreCase(s, substr))
}

func findIgnoreCase(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			sc, fc := s[i+j], substr[j]
			if sc >= 'A' && sc <= 'Z' {
				sc += 'a' - 'A'
			}
			if fc >= 'A' && fc <= 'Z' {
				fc += 'a' - 'A'
			}
			if sc != fc {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
