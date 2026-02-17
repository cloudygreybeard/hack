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
	"os/exec"
	"path/filepath"

	"github.com/cloudygreybeard/hack/internal/config"
	"github.com/cloudygreybeard/hack/internal/log"
	"github.com/spf13/cobra"
)

var (
	editForceTerminal bool
	editForceIDE      bool
)

var editCmd = &cobra.Command{
	Use:     "edit <filter>",
	Aliases: []string{"e", "open"},
	Short:   "Open a workspace in an editor or IDE",
	Long: `Find a matching workspace and open it for editing.

The edit behavior is determined by the edit_mode configuration:

  auto      Use IDE if configured, otherwise terminal editor (default)
  terminal  Open terminal editor (e.g. vim) on README.md
  ide       Launch IDE (e.g. cursor, code) on the workspace directory

Use --terminal or --ide to override for a single invocation.

Examples:
  hack edit my-project          # Open per edit_mode config
  hack edit my-project -t       # Force terminal editor
  hack edit my-project --ide    # Force IDE
  hack open my-project          # Alias for edit`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
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

		forceMode := ""
		if editForceTerminal {
			forceMode = "terminal"
		} else if editForceIDE {
			forceMode = "ide"
		}
		openWorkspace(dir, forceMode)
	},
}

func init() {
	rootCmd.AddCommand(editCmd)

	editCmd.Flags().BoolVarP(&editForceTerminal, "terminal", "t", false, "force terminal editor")
	editCmd.Flags().BoolVar(&editForceIDE, "ide", false, "force IDE")
}

// openWorkspace opens a workspace directory using the configured edit mode.
// forceMode overrides the configured edit_mode if non-empty.
func openWorkspace(dir, forceMode string) {
	mode := config.C.EditMode
	if forceMode != "" {
		mode = forceMode
	}

	if mode == "auto" {
		if config.C.IDE != "" {
			mode = "ide"
		} else {
			mode = "terminal"
		}
	}

	log.Debug("edit mode: %s", mode)

	switch mode {
	case "ide":
		openIDE(dir)
	case "terminal":
		openTerminalEditor(dir)
	default:
		log.Warn("unknown edit_mode %q, falling back to terminal", mode)
		openTerminalEditor(dir)
	}
}

// openIDE launches an IDE on the workspace directory.
// The IDE process is started in the background (non-blocking).
func openIDE(dir string) {
	ide := config.C.IDE
	if ide == "" {
		log.Warn("no IDE configured, falling back to terminal editor")
		openTerminalEditor(dir)
		return
	}

	log.Debug("launching IDE: %s %s", ide, dir)
	cmd := exec.Command(ide, ".")
	cmd.Dir = dir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Error("launching IDE: %v", err)
		return
	}
	// Don't wait for the IDE process to complete
	go func() {
		_ = cmd.Wait()
	}()
}

// openTerminalEditor opens a terminal editor on README.md in the given directory.
// Requires a TTY on stdin.
func openTerminalEditor(dir string) {
	stdinInfo, err := os.Stdin.Stat()
	if err != nil || (stdinInfo.Mode()&os.ModeCharDevice) == 0 {
		log.Debug("skipping terminal editor: stdin is not a terminal")
		return
	}

	readmePath := filepath.Join(dir, "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		log.Debug("skipping terminal editor: README.md does not exist")
		return
	}

	editor := config.C.Editor
	log.Debug("opening terminal editor: %s %s", editor, readmePath)
	cmd := exec.Command(editor, readmePath)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}
