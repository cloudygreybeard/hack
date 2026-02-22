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

	"github.com/cloudygreybeard/hack/internal/config"
	"github.com/cloudygreybeard/hack/internal/log"
	"github.com/spf13/cobra"
)

var rmForce bool

var rmCmd = &cobra.Command{
	Use:     "rm <filter>",
	Aliases: []string{"remove", "delete"},
	Short:   "Remove a hack workspace",
	Long: `Remove a hack workspace matching the given filter.

Requires confirmation unless --force is specified.

Examples:
  hack rm old-experiment         # Remove matching workspace (with confirmation)
  hack rm old-experiment --force # Remove without confirmation`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeWorkspaces,
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

		dirName := filepath.Base(dir)

		if !rmForce {
			fmt.Fprintf(os.Stderr, "remove workspace %q? [y/N] ", dirName)
			var response string
			if _, err := fmt.Scanln(&response); err != nil || (response != "y" && response != "Y") {
				fmt.Fprintln(os.Stderr, "cancelled")
				return
			}
		}

		log.Verbose("removing workspace: %s", dir)
		if err := os.RemoveAll(dir); err != nil {
			log.Error("removing workspace: %v", err)
			os.Exit(1)
		}

		fmt.Fprintf(os.Stderr, "removed: %s\n", dirName)
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)

	rmCmd.Flags().BoolVarP(&rmForce, "force", "f", false, "skip confirmation prompt")
}
