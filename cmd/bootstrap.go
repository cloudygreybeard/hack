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
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	bootstrapShell   string
	bootstrapInstall bool
	bootstrapRcFile  string
)

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Output or install shell integration",
	Long: `Output or install a shell function that enables directory changing.

The snippet wraps the hack command to automatically cd into directories
when the command outputs a valid path.

By default, prints the snippet to stdout. Use --install to append it
to your shell's rc file.

Examples:
  hack bootstrap                    # Print snippet (auto-detect shell)
  hack bootstrap --shell bash       # Print bash snippet
  hack bootstrap --install          # Install to detected rc file
  hack bootstrap --install --rc ~/.bashrc  # Install to specific file`,
	Run: func(cmd *cobra.Command, args []string) {
		shell := bootstrapShell
		if shell == "" {
			shell = detectShell()
		}

		snippet := getSnippet(shell)

		if bootstrapInstall {
			rcFile := bootstrapRcFile
			if rcFile == "" {
				rcFile = getRcFile(shell)
			}
			if err := installSnippet(rcFile, snippet, shell); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Println(snippet)
		}
	},
}

func init() {
	rootCmd.AddCommand(bootstrapCmd)

	bootstrapCmd.Flags().StringVar(&bootstrapShell, "shell", "", "shell type (bash, zsh, fish)")
	bootstrapCmd.Flags().BoolVar(&bootstrapInstall, "install", false, "install snippet to rc file")
	bootstrapCmd.Flags().StringVar(&bootstrapRcFile, "rc", "", "rc file path (default: auto-detect)")
}

func detectShell() string {
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		return "bash"
	}
	return filepath.Base(strings.TrimSuffix(shellPath, filepath.Ext(shellPath)))
}

func getSnippet(shell string) string {
	switch shell {
	case "fish":
		return fishSnippet
	case "zsh", "bash", "sh":
		return bashSnippet
	default:
		fmt.Fprintf(os.Stderr, "unknown shell %q, using bash-compatible syntax\n", shell)
		return bashSnippet
	}
}

func getRcFile(shell string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}

	switch shell {
	case "fish":
		return filepath.Join(home, ".config", "fish", "config.fish")
	case "zsh":
		return filepath.Join(home, ".zshrc")
	case "bash":
		// Prefer .bashrc, fall back to .bash_profile for login shells
		bashrc := filepath.Join(home, ".bashrc")
		if _, err := os.Stat(bashrc); err == nil {
			return bashrc
		}
		return filepath.Join(home, ".bash_profile")
	default:
		return filepath.Join(home, ".bashrc")
	}
}

func installSnippet(rcFile, snippet, shell string) error {
	// Expand ~ if present
	if strings.HasPrefix(rcFile, "~/") {
		home, _ := os.UserHomeDir()
		rcFile = filepath.Join(home, rcFile[2:])
	}

	// Check if already installed
	if isSnippetInstalled(rcFile) {
		fmt.Printf("hack shell integration already present in %s\n", rcFile)
		return nil
	}

	// Ensure parent directory exists (for fish)
	if err := os.MkdirAll(filepath.Dir(rcFile), 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// Back up existing file
	if _, err := os.Stat(rcFile); err == nil {
		backupFile := rcFile + ".backup." + time.Now().Format("20060102-150405")
		data, err := os.ReadFile(rcFile)
		if err != nil {
			return fmt.Errorf("reading rc file: %w", err)
		}
		if err := os.WriteFile(backupFile, data, 0644); err != nil {
			return fmt.Errorf("creating backup: %w", err)
		}
		fmt.Printf("backed up %s to %s\n", rcFile, backupFile)
	}

	// Open file for appending
	f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening rc file: %w", err)
	}
	defer f.Close()

	// Add newlines before snippet if file is not empty
	info, _ := f.Stat()
	if info.Size() > 0 {
		if _, err := f.WriteString("\n\n"); err != nil {
			return fmt.Errorf("writing to rc file: %w", err)
		}
	}

	// Write snippet
	if _, err := f.WriteString(snippet + "\n"); err != nil {
		return fmt.Errorf("writing snippet: %w", err)
	}

	fmt.Printf("installed hack shell integration to %s\n", rcFile)
	fmt.Printf("restart your shell or run: source %s\n", rcFile)
	return nil
}

func isSnippetInstalled(rcFile string) bool {
	f, err := os.Open(rcFile)
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		// Check for the distinctive marker
		if strings.Contains(line, "# hack shell integration") {
			return true
		}
		// Also check for the function definition
		if strings.Contains(line, "hack()") || strings.Contains(line, "function hack") {
			return true
		}
	}
	return false
}

const bashSnippet = `# hack shell integration
hack() {
    local cd_target
    local _hack_fd=63
    # Save stdout to temp fd, capture cd target fd, restore stdout
    eval "exec $_hack_fd>&1"
    cd_target=$(HACK_CD_FD=$_hack_fd command hack "$@" $_hack_fd>&1 1>&$_hack_fd 2>&1)
    local exit_code=$?
    eval "exec $_hack_fd>&-"

    if [[ $exit_code -eq 0 && -n "$cd_target" && -d "$cd_target" ]]; then
        cd "$cd_target"
    fi
    return $exit_code
}`

const fishSnippet = `# hack shell integration
function hack
    set -l _hack_fd 63
    # Capture cd target on high fd, leave stdout/stderr for interactive use
    set -l cd_target (begin; env HACK_CD_FD=$_hack_fd command hack $argv $_hack_fd>&1 1>&$_hack_fd 2>&1; end)
    set -l exit_code $status

    if test $exit_code -eq 0 -a -n "$cd_target" -a -d "$cd_target"
        cd "$cd_target"
    end
    return $exit_code
end`
