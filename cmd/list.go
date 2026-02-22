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
	"strings"

	"github.com/cloudygreybeard/hack/internal/config"
	"github.com/cloudygreybeard/hack/internal/workspace"
	"github.com/spf13/cobra"
)

var (
	listSelector   string
	listShowLabels bool
)

var listCmd = &cobra.Command{
	Use:     "list [filter]",
	Aliases: []string{"l", "ls", "get", "g"},
	Short:   "List hack workspaces",
	Long: `List hack workspaces in the root directory.

Optionally filter by a substring (case-insensitive match).
Use -l to filter by labels (comma-separated key=value, AND semantics).

Examples:
  hack list                        # List all workspaces
  hack list api                    # List workspaces containing "api"
  hack list -l domain=aro          # List workspaces with label domain=aro
  hack list -l domain=aro,lang=go  # Multiple label match (AND)
  hack list --show-labels          # Show labels alongside workspace names
  hack ls                          # Short alias`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeWorkspaces,
	Run: func(cmd *cobra.Command, args []string) {
		filter := ""
		if len(args) > 0 {
			filter = strings.ToLower(args[0])
		}

		var selector map[string]string
		if listSelector != "" {
			var err error
			selector, err = workspace.ParseSelector(listSelector)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
		}

		entries, err := os.ReadDir(config.C.RootDir)
		if err != nil {
			if os.IsNotExist(err) {
				return
			}
			fmt.Fprintf(os.Stderr, "error reading directory: %v\n", err)
			os.Exit(1)
		}

		type dirEntry struct {
			name   string
			labels map[string]string
		}

		var dirs []dirEntry
		for _, entry := range entries {
			if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			name := entry.Name()
			if filter != "" && !strings.Contains(strings.ToLower(name), filter) {
				continue
			}

			var labels map[string]string
			if selector != nil || listShowLabels {
				meta, _ := workspace.Load(filepath.Join(config.C.RootDir, name))
				labels = meta.MetadataObj.Labels

				if selector != nil && !workspace.MatchesSelector(meta, selector) {
					continue
				}
			}

			dirs = append(dirs, dirEntry{name: name, labels: labels})
		}

		sort.Slice(dirs, func(i, j int) bool {
			return dirs[i].name < dirs[j].name
		})

		for _, d := range dirs {
			if listShowLabels && len(d.labels) > 0 {
				fmt.Printf("%s  %s\n", d.name, workspace.FormatLabels(d.labels))
			} else {
				fmt.Println(d.name)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringVarP(&listSelector, "selector", "l", "", "label selector (key=value,key2=value2)")
	listCmd.Flags().BoolVar(&listShowLabels, "show-labels", false, "show labels alongside workspace names")
}
