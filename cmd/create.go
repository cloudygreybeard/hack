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
	"strings"
	"time"

	"github.com/cloudygreybeard/hack/internal/config"
	"github.com/cloudygreybeard/hack/internal/log"
	"github.com/cloudygreybeard/hack/internal/pattern"
	"github.com/cloudygreybeard/hack/internal/prompt"
	"github.com/cloudygreybeard/hack/internal/security"
	"github.com/spf13/cobra"
)

var (
	createNoGit    bool
	createNoReadme bool
	createNoEdit   bool
	createPattern  string
	createModule   string
	createAppName  string
)

var createCmd = &cobra.Command{
	Use:     "create <workspace-name>",
	Aliases: []string{"c", "new", "edit", "e"},
	Short:   "Create a new hack workspace or add an app to existing one",
	Long: `Create a hack workspace with today's date prefix.

The workspace directory is created in the format: YYYY-MM-DD.<workspace-name>

Without a pattern, creates an empty workspace with just a README.md.
With a pattern (-p), creates an app subdirectory inside the workspace.

Use -a/--app-name to specify a different name for the app directory
(defaults to the workspace name).

If the workspace already exists and a pattern is specified, the app is
added without overwriting workspace-level files.

Examples:
  hack create my-project               # Empty workspace with README.md
  hack create my-project -p go-cli     # Workspace with my-project/ app inside
  hack create my-project -p go-cli -a myapp  # Workspace with myapp/ app inside
  hack create my-project -p go-cli -a other  # Add another app to existing workspace
  hack create my-project -p go-cli -i  # Interactive mode for variables
  hack create foo --no-git             # Skip git initialization`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		workspaceName := args[0]
		datePrefix := time.Now().Format("2006-01-02")

		// Validate workspace name
		if err := security.ValidateName(workspaceName); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		// App name defaults to workspace name
		appName := workspaceName
		if createAppName != "" {
			// Validate app name if explicitly provided
			if err := security.ValidateName(createAppName); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			appName = createAppName
		}

		log.Debug("workspace: %s, app: %s, date: %s", workspaceName, appName, datePrefix)

		dirName := fmt.Sprintf("%s.%s", datePrefix, workspaceName)
		fullPath := filepath.Join(config.C.RootDir, dirName)

		workspaceExists := false
		if _, err := os.Stat(fullPath); err == nil {
			workspaceExists = true
		}

		// If workspace exists, we're adding an app
		addMode := workspaceExists

		if !workspaceExists {
			log.Verbose("creating new workspace: %s", dirName)

			// Ensure root directory exists
			if err := os.MkdirAll(config.C.RootDir, 0755); err != nil {
				log.Error("creating root directory: %v", err)
				os.Exit(1)
			}

			// Create the workspace directory
			if err := os.MkdirAll(fullPath, 0755); err != nil {
				log.Error("creating directory: %v", err)
				os.Exit(1)
			}
			log.Debug("created directory: %s", fullPath)
		} else {
			log.Verbose("using existing workspace: %s", dirName)
		}

		// Apply pattern if specified
		if createPattern != "" {
			log.Verbose("applying pattern: %s", createPattern)
			if err := applyPatternWithPrompt(workspaceName, appName, datePrefix, fullPath, addMode); err != nil {
				log.Error("applying pattern: %v", err)
				os.Exit(1)
			}
		}

		// Initialize git if enabled (only for new workspaces)
		if !workspaceExists && config.C.GitInit && !createNoGit {
			log.Debug("initializing git repository")
			if err := initGit(fullPath); err != nil {
				log.Warn("git init failed: %v", err)
			}
		}

		// Create README.md if enabled and pattern didn't create one (only for new workspaces)
		readmePath := filepath.Join(fullPath, "README.md")
		if !workspaceExists && config.C.CreateReadme && !createNoReadme {
			if _, err := os.Stat(readmePath); os.IsNotExist(err) {
				content := fmt.Sprintf("# %s\n\n", workspaceName)
				if err := os.WriteFile(readmePath, []byte(content), 0644); err != nil {
					log.Warn("failed to create README.md: %v", err)
				} else {
					log.FileCreated("README.md")
				}
			}
		}

		// Output the path (to fd 3 if available, otherwise stdout)
		outputCdTarget(fullPath)

		// Open editor if not disabled
		if !createNoEdit {
			openEditor(fullPath)
		}
	},
}

func init() {
	rootCmd.AddCommand(createCmd)

	createCmd.Flags().BoolVar(&createNoGit, "no-git", false, "skip git initialization")
	createCmd.Flags().BoolVar(&createNoReadme, "no-readme", false, "skip README.md creation")
	createCmd.Flags().BoolVar(&createNoEdit, "no-edit", false, "don't open editor after creation")
	createCmd.Flags().BoolVarP(&createNoEdit, "N", "N", false, "don't open editor (short form)")
	createCmd.Flags().StringVarP(&createPattern, "pattern", "p", "", "apply pattern to create app")
	createCmd.Flags().StringVarP(&createModule, "module", "m", "", "Go module path (default: example.com/<app>)")
	createCmd.Flags().StringVarP(&createAppName, "app-name", "a", "", "app directory name (default: <workspace-name>)")
}

func applyPatternWithPrompt(workspaceName, appName, datePrefix, fullPath string, addMode bool) error {
	// Load pattern metadata
	patternPath := filepath.Join(config.C.PatternsDir, createPattern)
	p, err := pattern.Load(patternPath)
	if err != nil {
		return fmt.Errorf("loading pattern: %w", err)
	}

	// Build default variables
	module := createModule
	if module == "" {
		if config.C.DefaultOrg != "" {
			module = fmt.Sprintf("github.com/%s/%s", config.C.DefaultOrg, appName)
		} else {
			module = fmt.Sprintf("example.com/%s", appName)
		}
	}

	vars := map[string]string{
		"name":     workspaceName,
		"app_name": appName,
		"Name":     toTitle(appName),
		"year":     time.Now().Format("2006"),
		"date":     datePrefix,
		"module":   module,
	}

	// Apply pattern-defined defaults for variables not already set
	for _, v := range p.Variables {
		if _, ok := vars[v.Name]; !ok && v.Default != "" {
			vars[v.Name] = v.Default
		}
	}

	// Interactive mode: prompt for additional variables
	if config.C.Interactive && len(p.Variables) > 0 {
		vars, err = prompt.PatternVariables(p, vars)
		if err != nil {
			return fmt.Errorf("prompting for variables: %w", err)
		}
	}

	// Apply the pattern
	opts := pattern.ApplyOptions{
		SkipExisting: addMode, // In add mode, don't overwrite existing files
	}
	return pattern.ApplyWithOptions(config.C.PatternsDir, createPattern, fullPath, vars, opts)
}

func initGit(dir string) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func openEditor(dir string) {
	// Only open editor if stdin is a terminal
	stdinInfo, err := os.Stdin.Stat()
	if err != nil || (stdinInfo.Mode()&os.ModeCharDevice) == 0 {
		log.Debug("skipping editor: stdin is not a terminal")
		return
	}

	readmePath := filepath.Join(dir, "README.md")

	// Check if README exists
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		log.Debug("skipping editor: README.md does not exist")
		return
	}

	log.Debug("opening editor: %s %s", config.C.Editor, readmePath)
	cmd := exec.Command(config.C.Editor, readmePath)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}

// toTitle converts a hyphenated name to TitleCase for use in class names.
// e.g., "my-tool" -> "MyTool"
func toTitle(s string) string {
	parts := strings.Split(s, "-")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}
