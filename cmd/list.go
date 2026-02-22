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
	"sort"
	"strings"

	"github.com/cloudygreybeard/hack/internal/config"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list [filter]",
	Aliases: []string{"l", "ls", "get", "g"},
	Short:   "List hack workspaces",
	Long: `List hack workspaces in the root directory.

Optionally filter by a substring (case-insensitive match).

Examples:
  hack list           # List all workspaces
  hack list api       # List workspaces containing "api"
  hack ls             # Short alias`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeWorkspaces,
	Run: func(cmd *cobra.Command, args []string) {
		filter := ""
		if len(args) > 0 {
			filter = strings.ToLower(args[0])
		}

		entries, err := os.ReadDir(config.C.RootDir)
		if err != nil {
			if os.IsNotExist(err) {
				return
			}
			fmt.Fprintf(os.Stderr, "error reading directory: %v\n", err)
			os.Exit(1)
		}

		var dirs []string
		for _, entry := range entries {
			if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			name := entry.Name()
			if filter != "" && !strings.Contains(strings.ToLower(name), filter) {
				continue
			}
			dirs = append(dirs, name)
		}

		sort.Strings(dirs)
		for _, dir := range dirs {
			fmt.Println(dir)
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
