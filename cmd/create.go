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
	"github.com/cloudygreybeard/hack/internal/workspace"
	"github.com/spf13/cobra"
)

var (
	createNoGit    bool
	createNoReadme bool
	createNoEdit   bool
	createDryRun   bool
	createPattern  string
	createModule   string
	createAppName  string
	createLabels   []string
)

var createCmd = &cobra.Command{
	Use:     "create <workspace-name>",
	Aliases: []string{"c", "new"},
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
  hack create foo --no-git             # Skip git initialization
  hack create foo -p go-cli --label domain=tools --label lang=go
  hack create foo -p go-cli --dry-run  # Show what would be created`,
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
			if err := security.ValidateName(createAppName); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			appName = createAppName
		}

		log.Debug("workspace: %s, app: %s, date: %s", workspaceName, appName, datePrefix)

		dirName := fmt.Sprintf("%s.%s", datePrefix, workspaceName)
		fullPath := filepath.Join(config.C.RootDir, dirName)

		// Resolve pattern inheritance chain (if pattern specified)
		var resolved []pattern.ResolvedPattern
		if createPattern != "" {
			log.Verbose("resolving pattern: %s", createPattern)
			var err error
			resolved, err = pattern.Resolve(config.C.PatternsDir, createPattern)
			if err != nil {
				log.Error("resolving pattern: %v", err)
				os.Exit(1)
			}
		}

		// Dry-run: show what would be created and exit
		if createDryRun {
			fmt.Fprintf(os.Stderr, "dry-run: would create workspace %s\n", dirName)
			if len(resolved) > 0 {
				fmt.Fprintf(os.Stderr, "patterns to apply (in order):\n")
				for _, rp := range resolved {
					fmt.Fprintf(os.Stderr, "  %s (weight: %d, %s)\n", rp.Pattern.Name, rp.Pattern.Weight, rp.Source)
				}
			}
			if len(createLabels) > 0 {
				fmt.Fprintf(os.Stderr, "labels: %s\n", strings.Join(createLabels, ", "))
			}
			return
		}

		workspaceExists := false
		if _, err := os.Stat(fullPath); err == nil {
			workspaceExists = true
		}

		addMode := workspaceExists

		if !workspaceExists {
			log.Verbose("creating new workspace: %s", dirName)

			if err := os.MkdirAll(config.C.RootDir, 0755); err != nil {
				log.Error("creating root directory: %v", err)
				os.Exit(1)
			}

			if err := os.MkdirAll(fullPath, 0755); err != nil {
				log.Error("creating directory: %v", err)
				os.Exit(1)
			}
			log.Debug("created directory: %s", fullPath)
		} else {
			log.Verbose("using existing workspace: %s", dirName)
		}

		// Apply pattern chain if specified
		if len(resolved) > 0 {
			log.Verbose("applying %d pattern(s)", len(resolved))
			if err := applyPatternChain(resolved, workspaceName, appName, datePrefix, fullPath, addMode); err != nil {
				log.Error("applying patterns: %v", err)
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

		// Write workspace metadata (.hack.yaml)
		if !workspaceExists {
			meta := buildWorkspaceMetadata(workspaceName, resolved)
			if err := workspace.Save(fullPath, meta); err != nil {
				log.Warn("failed to write .hack.yaml: %v", err)
			} else {
				log.FileCreated(".hack.yaml")
			}
		}

		// Output the path (via HACK_CD_FD if available, otherwise stdout)
		outputCdTarget(fullPath)

		// Open editor/IDE if not disabled
		if !createNoEdit {
			openWorkspace(fullPath, "")
		}
	},
}

func init() {
	rootCmd.AddCommand(createCmd)

	createCmd.Flags().BoolVar(&createNoGit, "no-git", false, "skip git initialization")
	createCmd.Flags().BoolVar(&createNoReadme, "no-readme", false, "skip README.md creation")
	createCmd.Flags().BoolVar(&createNoEdit, "no-edit", false, "don't open editor after creation")
	createCmd.Flags().BoolVarP(&createNoEdit, "N", "N", false, "don't open editor (short form)")
	createCmd.Flags().BoolVar(&createDryRun, "dry-run", false, "show what would be created without writing files")
	createCmd.Flags().StringVarP(&createPattern, "pattern", "p", "", "apply pattern to create app")
	createCmd.Flags().StringVarP(&createModule, "module", "m", "", "Go module path (default: example.com/<app>)")
	createCmd.Flags().StringVarP(&createAppName, "app-name", "a", "", "app directory name (default: <workspace-name>)")
	createCmd.Flags().StringArrayVar(&createLabels, "label", nil, "set workspace label (key=value, repeatable)")

	_ = createCmd.RegisterFlagCompletionFunc("pattern", completePatterns)
}

// applyPatternChain applies a resolved chain of patterns in order.
// The last pattern in the chain is the one requested; earlier ones are inherited.
func applyPatternChain(resolved []pattern.ResolvedPattern, workspaceName, appName, datePrefix, fullPath string, addMode bool) error {
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

	// Collect variable defaults from all patterns in the chain
	for _, rp := range resolved {
		for _, v := range rp.Pattern.Variables {
			if _, ok := vars[v.Name]; !ok && v.Default != "" {
				vars[v.Name] = v.Default
			}
		}
	}

	// Interactive mode: prompt using the requested pattern's variables
	if config.C.Interactive {
		requested := resolved[len(resolved)-1].Pattern
		if len(requested.Variables) > 0 {
			var err error
			vars, err = prompt.PatternVariables(requested, vars)
			if err != nil {
				return fmt.Errorf("prompting for variables: %w", err)
			}
		}
	}

	// Apply each pattern in order
	for i, rp := range resolved {
		opts := pattern.ApplyOptions{
			SkipExisting: addMode && i == 0,
		}
		log.Verbose("applying pattern %s (%s)", rp.Pattern.Name, rp.Source)
		if err := pattern.ApplyWithOptions(config.C.PatternsDir, rp.Pattern.Name, fullPath, vars, opts); err != nil {
			return fmt.Errorf("applying %s: %w", rp.Pattern.Name, err)
		}
	}

	// Run post-create hooks from each pattern in order
	for _, rp := range resolved {
		if err := pattern.RunPostCreate(rp.Pattern, fullPath, vars); err != nil {
			return fmt.Errorf("post-create hooks for %s: %w", rp.Pattern.Name, err)
		}
	}

	return nil
}

// buildWorkspaceMetadata constructs the initial .hack.yaml for a new workspace.
func buildWorkspaceMetadata(workspaceName string, resolved []pattern.ResolvedPattern) workspace.Metadata {
	labels := make(map[string]string)
	annotations := make(map[string]string)

	// Apply default labels from patterns (earlier patterns first, later override)
	for _, rp := range resolved {
		for k, v := range rp.Pattern.DefaultLabels {
			labels[k] = v
		}
	}

	// Record the pattern used as an annotation (provenance, not for selection)
	if len(resolved) > 0 {
		requested := resolved[len(resolved)-1]
		annotations["hack.dev/pattern"] = requested.Pattern.Name
	}

	// Apply user-specified labels (highest priority)
	for _, l := range createLabels {
		key, value, _, err := workspace.ParseLabelArg(l)
		if err != nil {
			log.Warn("invalid label %q: %v", l, err)
			continue
		}
		labels[key] = value
	}

	return workspace.Metadata{
		APIVersion: "hack/v1",
		Kind:       "Workspace",
		MetadataObj: workspace.MetadataFields{
			Name:        workspaceName,
			Labels:      labels,
			Annotations: annotations,
		},
	}
}

func initGit(dir string) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
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
