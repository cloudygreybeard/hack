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
	"github.com/cloudygreybeard/hack/internal/pattern"
	"github.com/cloudygreybeard/hack/internal/prompt"
	"github.com/spf13/cobra"
)

var (
	createNoGit    bool
	createNoReadme bool
	createNoEdit   bool
	createPattern  string
	createModule   string
)

var createCmd = &cobra.Command{
	Use:     "create <name>",
	Aliases: []string{"c", "new", "edit", "e"},
	Short:   "Create a new hack workspace",
	Long: `Create a new hack workspace with today's date prefix.

The directory is created in the format: YYYY-MM-DD.<name>
By default, it will:
  - Initialize a git repository
  - Create a README.md file
  - Open the README.md in your editor

Use -p/--pattern to apply a project pattern after creation.
Use -i/--interactive to prompt for pattern variables.

Examples:
  hack create my-project             # Creates ~/hack/2026-01-26.my-project
  hack create my-project -p hello    # Create with pattern
  hack create my-project -p hello -i # Interactive mode for variables
  hack create foo --no-git           # Skip git initialization`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		datePrefix := time.Now().Format("2006-01-02")
		dirName := fmt.Sprintf("%s.%s", datePrefix, name)
		fullPath := filepath.Join(config.C.RootDir, dirName)

		// Check if directory already exists
		if _, err := os.Stat(fullPath); err == nil {
			fmt.Println(fullPath)
			if !createNoEdit {
				openEditor(fullPath)
			}
			return
		}

		// Ensure root directory exists
		if err := os.MkdirAll(config.C.RootDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "error creating root directory: %v\n", err)
			os.Exit(1)
		}

		// Create the hack directory
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "error creating directory: %v\n", err)
			os.Exit(1)
		}

		// Apply pattern if specified
		if createPattern != "" {
			if err := applyPatternWithPrompt(name, datePrefix, fullPath); err != nil {
				fmt.Fprintf(os.Stderr, "warning: pattern failed: %v\n", err)
			}
		}

		// Initialize git if enabled
		if config.C.GitInit && !createNoGit {
			if err := initGit(fullPath); err != nil {
				fmt.Fprintf(os.Stderr, "warning: git init failed: %v\n", err)
			}
		}

		// Create README.md if enabled and pattern didn't create one
		readmePath := filepath.Join(fullPath, "README.md")
		if config.C.CreateReadme && !createNoReadme {
			if _, err := os.Stat(readmePath); os.IsNotExist(err) {
				content := fmt.Sprintf("# %s\n\n", name)
				if err := os.WriteFile(readmePath, []byte(content), 0644); err != nil {
					fmt.Fprintf(os.Stderr, "warning: failed to create README.md: %v\n", err)
				}
			}
		}

		// Output the path
		fmt.Println(fullPath)

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
	createCmd.Flags().StringVarP(&createPattern, "pattern", "p", "", "apply pattern after creation")
	createCmd.Flags().StringVarP(&createModule, "module", "m", "", "Go module path (default: example.com/<name>)")
}

func applyPatternWithPrompt(name, datePrefix, fullPath string) error {
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
			module = fmt.Sprintf("github.com/%s/%s", config.C.DefaultOrg, name)
		} else {
			module = fmt.Sprintf("example.com/%s", name)
		}
	}

	vars := map[string]string{
		"name":   name,
		"Name":   toTitle(name),
		"year":   time.Now().Format("2006"),
		"date":   datePrefix,
		"module": module,
	}

	// Interactive mode: prompt for additional variables
	if config.C.Interactive && len(p.Variables) > 0 {
		vars, err = prompt.PatternVariables(p, vars)
		if err != nil {
			return fmt.Errorf("prompting for variables: %w", err)
		}
	}

	// Apply the pattern
	return pattern.Apply(config.C.PatternsDir, createPattern, fullPath, vars)
}

func initGit(dir string) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func openEditor(dir string) {
	readmePath := filepath.Join(dir, "README.md")

	// Check if README exists
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		return
	}

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
