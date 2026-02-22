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

	"github.com/cloudygreybeard/hack/internal/config"
	"github.com/cloudygreybeard/hack/internal/log"
	"github.com/spf13/cobra"
)

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage hack plugins",
	Long: `Manage hack plugins.

Plugins are executable files in ~/.hack/plugins/ that extend hack with
custom commands. Each executable becomes a subcommand of hack.

For example, an executable at ~/.hack/plugins/hack-deploy becomes
available as 'hack deploy'.

Plugin naming convention:
  hack-<name>     Becomes 'hack <name>'
  <name>          Becomes 'hack <name>'

Plugins receive all remaining arguments and the following environment
variables:
  HACK_ROOT_DIR      Root directory for workspaces
  HACK_PATTERNS_DIR  Patterns directory
  HACK_PLUGINS_DIR   Plugins directory`,
}

var pluginListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List installed plugins",
	Run: func(cmd *cobra.Command, args []string) {
		plugins, err := discoverPlugins(config.C.PluginsDir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Fprintln(os.Stderr, "no plugins directory")
				fmt.Fprintf(os.Stderr, "create %s and add executables to extend hack\n", config.C.PluginsDir)
				return
			}
			fmt.Fprintf(os.Stderr, "error listing plugins: %v\n", err)
			os.Exit(1)
		}

		if len(plugins) == 0 {
			fmt.Fprintln(os.Stderr, "no plugins installed")
			fmt.Fprintf(os.Stderr, "add executables to %s to extend hack\n", config.C.PluginsDir)
			return
		}

		for _, p := range plugins {
			fmt.Printf("%-15s %s\n", p.name, p.path)
		}
	},
}

type pluginInfo struct {
	name string
	path string
}

// discoverPlugins scans the plugins directory for executables.
func discoverPlugins(pluginsDir string) ([]pluginInfo, error) {
	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		return nil, err
	}

	var plugins []pluginInfo
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		path := filepath.Join(pluginsDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Check if executable
		if info.Mode()&0111 == 0 {
			continue
		}

		name := entry.Name()
		name = strings.TrimPrefix(name, "hack-")

		plugins = append(plugins, pluginInfo{name: name, path: path})
	}

	return plugins, nil
}

// RegisterPlugins discovers plugins and adds them as Cobra commands.
// Called during init so plugin commands appear in help and completion.
func RegisterPlugins() {
	_ = config.Init()
	plugins, err := discoverPlugins(config.C.PluginsDir)
	if err != nil {
		return
	}

	for _, p := range plugins {
		pluginPath := p.path
		pluginName := p.name
		cmd := &cobra.Command{
			Use:                pluginName,
			Short:              fmt.Sprintf("Plugin: %s", pluginName),
			DisableFlagParsing: true,
			Run: func(cmd *cobra.Command, args []string) {
				runPlugin(pluginPath, args)
			},
		}
		rootCmd.AddCommand(cmd)
		log.Debug("registered plugin: %s -> %s", pluginName, pluginPath)
	}
}

func runPlugin(pluginPath string, args []string) {
	cmd := exec.Command(pluginPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Env = append(os.Environ(),
		"HACK_ROOT_DIR="+config.C.RootDir,
		"HACK_PATTERNS_DIR="+config.C.PatternsDir,
		"HACK_PLUGINS_DIR="+config.C.PluginsDir,
	)

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "plugin error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(pluginCmd)
	pluginCmd.AddCommand(pluginListCmd)
}
