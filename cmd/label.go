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

	"github.com/cloudygreybeard/hack/internal/config"
	"github.com/cloudygreybeard/hack/internal/log"
	"github.com/cloudygreybeard/hack/internal/workspace"
	"github.com/spf13/cobra"
)

var labelList bool

var labelCmd = &cobra.Command{
	Use:   "label <filter> [key=value ...] [key- ...]",
	Short: "Manage workspace labels",
	Long: `Set, remove, or list labels on a workspace.

Labels are key-value pairs stored in .hack.yaml, used for filtering
and selection (similar to Kubernetes labels).

Set labels:
  hack label my-project domain=aro lang=go

Remove labels (trailing dash, like kubectl):
  hack label my-project domain-

List labels:
  hack label my-project --list`,
	Args:              cobra.MinimumNArgs(1),
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

		meta, err := workspace.Load(dir)
		if err != nil {
			log.Error("loading metadata: %v", err)
			os.Exit(1)
		}

		if labelList || len(args) == 1 {
			if len(meta.MetadataObj.Labels) == 0 {
				fmt.Fprintln(os.Stderr, "no labels set")
				return
			}
			keys := make([]string, 0, len(meta.MetadataObj.Labels))
			for k := range meta.MetadataObj.Labels {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Printf("%s=%s\n", k, meta.MetadataObj.Labels[k])
			}
			return
		}

		if meta.MetadataObj.Labels == nil {
			meta.MetadataObj.Labels = make(map[string]string)
		}

		for _, arg := range args[1:] {
			key, value, remove, err := workspace.ParseLabelArg(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			if remove {
				delete(meta.MetadataObj.Labels, key)
				log.Verbose("removed label: %s", key)
			} else {
				meta.MetadataObj.Labels[key] = value
				log.Verbose("set label: %s=%s", key, value)
			}
		}

		if err := workspace.Save(dir, meta); err != nil {
			log.Error("saving metadata: %v", err)
			os.Exit(1)
		}
	},
}

var annotateCmd = &cobra.Command{
	Use:   "annotate <filter> [key=value ...] [key- ...]",
	Short: "Manage workspace annotations",
	Long: `Set, remove, or list annotations on a workspace.

Annotations are arbitrary key-value metadata stored in .hack.yaml.
Unlike labels, annotations are not used for filtering or selection.

Set annotations:
  hack annotate my-project hack.dev/jira=OCPBUGS-12345

Remove annotations:
  hack annotate my-project hack.dev/jira-

List annotations:
  hack annotate my-project --list`,
	Args:              cobra.MinimumNArgs(1),
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

		meta, err := workspace.Load(dir)
		if err != nil {
			log.Error("loading metadata: %v", err)
			os.Exit(1)
		}

		if annotateList || len(args) == 1 {
			if len(meta.MetadataObj.Annotations) == 0 {
				fmt.Fprintln(os.Stderr, "no annotations set")
				return
			}
			keys := make([]string, 0, len(meta.MetadataObj.Annotations))
			for k := range meta.MetadataObj.Annotations {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Printf("%s=%s\n", k, meta.MetadataObj.Annotations[k])
			}
			return
		}

		if meta.MetadataObj.Annotations == nil {
			meta.MetadataObj.Annotations = make(map[string]string)
		}

		for _, arg := range args[1:] {
			key, value, remove, err := workspace.ParseLabelArg(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			if remove {
				delete(meta.MetadataObj.Annotations, key)
				log.Verbose("removed annotation: %s", key)
			} else {
				meta.MetadataObj.Annotations[key] = value
				log.Verbose("set annotation: %s=%s", key, value)
			}
		}

		if err := workspace.Save(dir, meta); err != nil {
			log.Error("saving metadata: %v", err)
			os.Exit(1)
		}
	},
}

var annotateList bool

func init() {
	rootCmd.AddCommand(labelCmd)
	rootCmd.AddCommand(annotateCmd)

	labelCmd.Flags().BoolVar(&labelList, "list", false, "list labels for the workspace")
	annotateCmd.Flags().BoolVar(&annotateList, "list", false, "list annotations for the workspace")
}
