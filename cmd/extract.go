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

	"github.com/cloudygreybeard/hack/internal/config"
	"github.com/cloudygreybeard/hack/internal/log"
	"github.com/cloudygreybeard/hack/internal/pattern"
	"github.com/spf13/cobra"
)

var (
	extractOutput      string
	extractName        string
	extractAppOnly     string
	extractTemplatise  bool
	extractInstall     bool
	extractExclude     []string
)

var patternExtractCmd = &cobra.Command{
	Use:   "extract [filter]",
	Short: "Extract a reusable pattern from an existing workspace",
	Long: `Create a pattern template by reverse-scaffolding an existing workspace.

Walks the workspace file tree, optionally replacing concrete values
(workspace name, app name, module path, year) with template expressions,
and generates a pattern.yaml with inferred variables.

If no filter is given, uses the current directory (if it looks like a
hack workspace).

Examples:
  hack pattern extract my-project
  hack pattern extract my-project -n my-cli-pattern
  hack pattern extract my-project --app-only my-tool
  hack pattern extract my-project --no-templatise
  hack pattern extract my-project --install
  hack pattern extract -o ./extracted/`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeWorkspaces,
	Run: func(cmd *cobra.Command, args []string) {
		var srcDir string

		if len(args) > 0 {
			dir, err := findMatchingDir(config.C.RootDir, args[0])
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			if dir == "" {
				fmt.Fprintf(os.Stderr, "no directory matching %q found\n", args[0])
				os.Exit(1)
			}
			srcDir = dir
		} else {
			cwd, err := os.Getwd()
			if err != nil {
				log.Error("getting current directory: %v", err)
				os.Exit(1)
			}
			srcDir = cwd
		}

		opts := pattern.ExtractOptions{
			PatternName:  extractName,
			OutputDir:    extractOutput,
			AppOnly:      extractAppOnly,
			Templatise:   extractTemplatise,
			ExtraExclude: extractExclude,
		}

		// Default output dir
		if opts.OutputDir == "" && !extractInstall {
			name := extractName
			if name == "" {
				name = "extracted-pattern"
			}
			opts.OutputDir = name
		}

		if extractInstall {
			tmpDir, err := os.MkdirTemp("", "hack-extract-*")
			if err != nil {
				log.Error("creating temp directory: %v", err)
				os.Exit(1)
			}
			defer os.RemoveAll(tmpDir)
			opts.OutputDir = tmpDir
		}

		log.Verbose("extracting pattern from %s", srcDir)

		if err := pattern.Extract(srcDir, opts); err != nil {
			log.Error("extracting pattern: %v", err)
			os.Exit(1)
		}

		if extractInstall {
			if err := pattern.Install(opts.OutputDir, config.C.PatternsDir); err != nil {
				log.Error("installing extracted pattern: %v", err)
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "pattern installed to %s\n", config.C.PatternsDir)
		} else {
			fmt.Fprintf(os.Stderr, "pattern extracted to %s\n", opts.OutputDir)
		}
	},
}

func init() {
	patternCmd.AddCommand(patternExtractCmd)

	patternExtractCmd.Flags().StringVarP(&extractOutput, "output", "o", "", "output directory for extracted pattern")
	patternExtractCmd.Flags().StringVarP(&extractName, "name", "n", "", "pattern name (default: inferred)")
	patternExtractCmd.Flags().StringVar(&extractAppOnly, "app-only", "", "extract only the named app subdirectory")
	patternExtractCmd.Flags().BoolVar(&extractTemplatise, "templatise", true, "replace concrete values with template variables")
	patternExtractCmd.Flags().BoolVar(&extractInstall, "install", false, "install extracted pattern to ~/.hack/patterns/")
	patternExtractCmd.Flags().StringArrayVar(&extractExclude, "exclude", nil, "additional exclusion globs (repeatable)")
}
