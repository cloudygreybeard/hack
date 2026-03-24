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
	"regexp"
	"strings"

	"github.com/cloudygreybeard/hack/internal/config"
	"github.com/cloudygreybeard/hack/internal/cursor"
	"github.com/cloudygreybeard/hack/internal/log"
	"github.com/cloudygreybeard/hack/internal/security"
	"github.com/cloudygreybeard/hack/internal/workspace"
	"github.com/spf13/cobra"
)

var datePrefixRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}\.`)

var (
	renameForce    bool
	renameNoCursor bool
)

var renameCmd = &cobra.Command{
	Use:     "rename <filter> <new-name>",
	Aliases: []string{"mv"},
	Short:   "Rename a hack workspace",
	Long: `Rename a hack workspace directory, update .hack.yaml metadata,
and migrate Cursor IDE workspace storage so chat history follows
the rename.

If new-name includes a date prefix (YYYY-MM-DD.concept), it is used
as the full directory name. Otherwise the existing date prefix is
preserved and only the concept portion is replaced.

Cursor must be closed for workspace storage migration to succeed.
Use --no-cursor to skip Cursor migration entirely.

Examples:
  hack rename old-concept new-concept       # Preserve date prefix
  hack rename proj 2026-03-20.new-project   # Full rename with new date
  hack rename proj new-name --no-cursor     # Skip Cursor migration`,
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: completeWorkspaces,
	Run:               runRename,
}

func init() {
	rootCmd.AddCommand(renameCmd)
	renameCmd.Flags().BoolVarP(&renameForce, "force", "f", false, "skip confirmation prompt")
	renameCmd.Flags().BoolVar(&renameNoCursor, "no-cursor", false, "skip Cursor workspace storage migration")
}

func runRename(cmd *cobra.Command, args []string) {
	filter := args[0]
	newName := args[1]

	oldDir, err := findMatchingDir(config.C.RootDir, filter)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if oldDir == "" {
		fmt.Fprintf(os.Stderr, "no directory matching %q found\n", filter)
		os.Exit(1)
	}

	oldBase := filepath.Base(oldDir)

	newDirName := buildNewDirName(oldBase, newName)

	conceptName := newDirName
	if parts := strings.SplitN(newDirName, ".", 2); len(parts) == 2 && datePrefixRe.MatchString(newDirName) {
		conceptName = parts[1]
	}
	if err := security.ValidateName(conceptName); err != nil {
		fmt.Fprintf(os.Stderr, "invalid name: %v\n", err)
		os.Exit(1)
	}

	newDir := filepath.Join(config.C.RootDir, newDirName)

	if oldDir == newDir {
		fmt.Fprintln(os.Stderr, "old and new names are the same")
		os.Exit(1)
	}

	if _, err := os.Stat(newDir); err == nil {
		fmt.Fprintf(os.Stderr, "directory %q already exists\n", newDirName)
		os.Exit(1)
	}

	if !renameForce {
		fmt.Fprintf(os.Stderr, "rename %q -> %q? [y/N] ", oldBase, newDirName)
		var response string
		if _, err := fmt.Scanln(&response); err != nil || (response != "y" && response != "Y") {
			fmt.Fprintln(os.Stderr, "cancelled")
			return
		}
	}

	if !renameNoCursor {
		cursorRunning, workspaceOpen := cursor.CheckWorkspace(oldDir)
		if workspaceOpen {
			if !renameForce {
				fmt.Fprintln(os.Stderr, "this workspace is open in Cursor; close it first or use --force")
				os.Exit(1)
			}
			log.Warn("workspace is open in Cursor; active sessions will be terminated")
		} else if cursorRunning {
			log.Verbose("Cursor is running but this workspace is not open; proceeding")
		}

		log.Verbose("renaming: %s -> %s", oldBase, newDirName)

		if err := os.Rename(oldDir, newDir); err != nil {
			log.Error("renaming directory: %v", err)
			os.Exit(1)
		}

		if err := cursor.MigrateWorkspace(oldDir, newDir); err != nil {
			log.Warn("Cursor migration: %v", err)
		}
	} else {
		log.Verbose("renaming: %s -> %s (skipping Cursor migration)", oldBase, newDirName)

		if err := os.Rename(oldDir, newDir); err != nil {
			log.Error("renaming directory: %v", err)
			os.Exit(1)
		}
	}

	if workspace.HasMetadata(newDir) {
		metaName := conceptName
		if err := workspace.Rename(newDir, metaName); err != nil {
			log.Warn("updating .hack.yaml: %v", err)
		}
	}

	fmt.Fprintf(os.Stderr, "renamed: %s -> %s\n", oldBase, newDirName)
	outputCdTarget(newDir)
}

// buildNewDirName constructs the new directory name from the old base name
// and the user-provided new name. If the new name has a date prefix, it is
// used as-is. Otherwise the old date prefix is preserved.
func buildNewDirName(oldBase, newName string) string {
	if datePrefixRe.MatchString(newName) {
		return newName
	}

	if loc := datePrefixRe.FindStringIndex(oldBase); loc != nil {
		datePrefix := oldBase[:loc[1]]
		return datePrefix + newName
	}

	return newName
}
