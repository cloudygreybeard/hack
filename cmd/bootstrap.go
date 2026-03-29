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

	"github.com/cloudygreybeard/hack/internal/config"
	"github.com/spf13/cobra"
)

var (
	bootstrapShell   string
	bootstrapInstall bool
	bootstrapRcFile  string
	bootstrapPersona string
	bootstrapAlias   string
)

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Output or install shell integration",
	Long: `Output or install a shell function that enables directory changing.

The snippet wraps the hack command to automatically cd into directories
when the command outputs a valid path.

By default, prints the snippet to stdout. Use --install to append it
to your shell's rc file.

Use --persona to generate a snippet for a named persona (e.g. diversions).
The generated function calls the base hack binary with HACK_NAME set.

Examples:
  hack bootstrap                                 # Print snippet (base hack)
  hack bootstrap --persona diversions --alias hd  # Print diversions snippet
  hack bootstrap --install                        # Install to detected rc file
  hack bootstrap --install --rc ~/.bashrc         # Install to specific file`,
	Run: func(cmd *cobra.Command, args []string) {
		shell := bootstrapShell
		if shell == "" {
			shell = detectShell()
		}

		persona := bootstrapPersona
		if persona == "" {
			persona = config.C.Persona
		}

		alias := bootstrapAlias
		if alias == "" && persona != "" {
			alias = config.C.ShellAlias
		}

		snippet := buildSnippet(shell, persona, alias)

		if bootstrapInstall {
			rcFile := bootstrapRcFile
			if rcFile == "" {
				rcFile = getRcFile(shell)
			}
			funcName := "hack"
			if persona != "" {
				funcName = "hack-" + persona
			}
			evalLine := buildEvalLine(shell, persona, alias)
			if err := installSnippet(rcFile, evalLine, funcName); err != nil {
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
	bootstrapCmd.Flags().StringVar(&bootstrapPersona, "persona", "", "generate snippet for a named persona")
	bootstrapCmd.Flags().StringVar(&bootstrapAlias, "alias", "", "short alias for the shell function")
}

func detectShell() string {
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		return "bash"
	}
	return filepath.Base(strings.TrimSuffix(shellPath, filepath.Ext(shellPath)))
}

// buildSnippet generates the shell integration snippet for the given shell,
// persona, and optional alias.
func buildSnippet(shell, persona, alias string) string {
	switch shell {
	case "fish":
		return buildFishSnippet(persona, alias)
	case "zsh", "bash", "sh":
		return buildBashSnippet(persona, alias)
	default:
		fmt.Fprintf(os.Stderr, "unknown shell %q, using bash-compatible syntax\n", shell)
		return buildBashSnippet(persona, alias)
	}
}

func buildBashSnippet(persona, alias string) string {
	funcName := "hack"
	hackCmd := "command hack"
	envPreamble := "HACK_CD_FD=63"
	marker := "# hack shell integration"

	if persona != "" {
		funcName = "hack-" + persona
		hackCmd = "command hack"
		envPreamble = fmt.Sprintf("HACK_CD_FD=63 HACK_NAME=%s", persona)
		marker = fmt.Sprintf("# hack-%s shell integration", persona)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", marker)
	fmt.Fprintf(&b, "%s() {\n", funcName)
	fmt.Fprintf(&b, "    local cd_target\n")
	fmt.Fprintf(&b, "    exec 4>&1\n")
	fmt.Fprintf(&b, "    cd_target=$(%s %s \"$@\" 63>&1 1>&4)\n", envPreamble, hackCmd)
	fmt.Fprintf(&b, "    local exit_code=$?\n")
	fmt.Fprintf(&b, "    exec 4>&-\n")
	fmt.Fprintf(&b, "\n")
	fmt.Fprintf(&b, "    if [[ $exit_code -eq 0 && -n \"$cd_target\" && -d \"$cd_target\" ]]; then\n")
	fmt.Fprintf(&b, "        cd \"$cd_target\"\n")
	fmt.Fprintf(&b, "    fi\n")
	fmt.Fprintf(&b, "    return $exit_code\n")
	fmt.Fprintf(&b, "}")

	if alias != "" {
		fmt.Fprintf(&b, "\nalias %s='%s'", alias, funcName)
	}

	return b.String()
}

func buildFishSnippet(persona, alias string) string {
	funcName := "hack"
	hackCmd := "command hack"
	envPreamble := "HACK_CD_FD=63"
	marker := "# hack shell integration"

	if persona != "" {
		funcName = "hack-" + persona
		hackCmd = "command hack"
		envPreamble = fmt.Sprintf("HACK_CD_FD=63 HACK_NAME=%s", persona)
		marker = fmt.Sprintf("# hack-%s shell integration", persona)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", marker)
	fmt.Fprintf(&b, "function %s\n", funcName)
	fmt.Fprintf(&b, "    set -l cd_target (begin; env %s %s $argv 63>&1 1>&2; end 2>&1)\n", envPreamble, hackCmd)
	fmt.Fprintf(&b, "    set -l exit_code $status\n")
	fmt.Fprintf(&b, "\n")
	fmt.Fprintf(&b, "    if test $exit_code -eq 0 -a -n \"$cd_target\" -a -d \"$cd_target\"\n")
	fmt.Fprintf(&b, "        cd \"$cd_target\"\n")
	fmt.Fprintf(&b, "    end\n")
	fmt.Fprintf(&b, "    return $exit_code\n")
	fmt.Fprintf(&b, "end")

	if alias != "" {
		fmt.Fprintf(&b, "\nalias %s='%s'", alias, funcName)
	}

	return b.String()
}

// buildEvalLine generates the guarded eval one-liner for --install.
func buildEvalLine(shell, persona, alias string) string {
	var bootstrapArgs []string
	if persona != "" {
		bootstrapArgs = append(bootstrapArgs, "--persona "+persona)
	}
	if alias != "" {
		bootstrapArgs = append(bootstrapArgs, "--alias "+alias)
	}

	args := ""
	if len(bootstrapArgs) > 0 {
		args = " " + strings.Join(bootstrapArgs, " ")
	}

	funcName := "hack"
	if persona != "" {
		funcName = "hack-" + persona
	}

	switch shell {
	case "fish":
		return fmt.Sprintf("# %s shell integration\ncommand -q hack; and eval (hack bootstrap%s)", funcName, args)
	default:
		return fmt.Sprintf("# %s shell integration\ncommand -v hack >/dev/null && eval \"$(hack bootstrap%s)\"", funcName, args)
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
		bashrc := filepath.Join(home, ".bashrc")
		if _, err := os.Stat(bashrc); err == nil {
			return bashrc
		}
		return filepath.Join(home, ".bash_profile")
	default:
		return filepath.Join(home, ".bashrc")
	}
}

func installSnippet(rcFile, snippet, funcName string) error {
	if strings.HasPrefix(rcFile, "~/") {
		home, _ := os.UserHomeDir()
		rcFile = filepath.Join(home, rcFile[2:])
	}

	marker := fmt.Sprintf("# %s shell integration", funcName)
	funcDef := funcName + "()"
	if isSnippetInstalled(rcFile, marker, funcDef) {
		fmt.Printf("%s shell integration already present in %s\n", funcName, rcFile)
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(rcFile), 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

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

	f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening rc file: %w", err)
	}
	defer f.Close()

	info, _ := f.Stat()
	if info.Size() > 0 {
		if _, err := f.WriteString("\n\n"); err != nil {
			return fmt.Errorf("writing to rc file: %w", err)
		}
	}

	if _, err := f.WriteString(snippet + "\n"); err != nil {
		return fmt.Errorf("writing snippet: %w", err)
	}

	fmt.Printf("installed %s shell integration to %s\n", funcName, rcFile)
	fmt.Printf("restart your shell or run: source %s\n", rcFile)
	return nil
}

func isSnippetInstalled(rcFile, marker, funcDef string) bool {
	f, err := os.Open(rcFile)
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, marker) || strings.Contains(line, funcDef) {
			return true
		}
	}
	return false
}
