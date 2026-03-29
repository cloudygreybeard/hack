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

// Package config provides configuration management for hack.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config holds the application configuration.
type Config struct {
	RootDir      string `mapstructure:"root_dir"`
	PatternsDir  string `mapstructure:"patterns_dir"`
	PluginsDir   string `mapstructure:"plugins_dir"`
	Editor       string `mapstructure:"editor"`
	IDE          string `mapstructure:"ide"`
	EditMode     string `mapstructure:"edit_mode"`
	GitInit      bool   `mapstructure:"git_init"`
	CreateReadme bool   `mapstructure:"create_readme"`
	Interactive  bool   `mapstructure:"interactive"`
	DefaultOrg   string `mapstructure:"default_org"`
	ShellAlias   string `mapstructure:"shell_alias"`

	// Persona is the resolved persona name (empty for base "hack").
	// Not loaded from config; derived from $HACK_NAME or os.Args[0].
	Persona string `mapstructure:"-"`
}

// C is the global configuration instance.
var C Config

// BinaryName returns the effective binary name for this persona.
func (c *Config) BinaryName() string {
	if c.Persona != "" {
		return "hack-" + c.Persona
	}
	return "hack"
}

// PersonaName resolves the persona from $HACK_NAME or os.Args[0].
// Returns empty string for the base "hack" persona.
func PersonaName() string {
	if name := os.Getenv("HACK_NAME"); name != "" {
		return name
	}
	base := filepath.Base(os.Args[0])
	if after, ok := strings.CutPrefix(base, "hack-"); ok && after != "" {
		return after
	}
	return ""
}

// Init initializes the configuration from file, environment, and defaults.
func Init() error {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}

	persona := PersonaName()

	// Derive identity from persona
	configName := ".hack"          // e.g. ".hack" or ".hack-diversions"
	rootBase := "hack"             // e.g. "hack" or "hack-diversions"
	dotBase := ".hack"             // e.g. ".hack" or ".hack-diversions"
	envPrefix := "HACK"            // e.g. "HACK" or "HACK_DIVERSIONS"
	if persona != "" {
		configName = ".hack-" + persona
		rootBase = "hack-" + persona
		dotBase = ".hack-" + persona
		envPrefix = "HACK_" + strings.ToUpper(strings.ReplaceAll(persona, "-", "_"))
	}

	// Set defaults
	viper.SetDefault("root_dir", filepath.Join(home, rootBase))
	viper.SetDefault("patterns_dir", filepath.Join(home, dotBase, "patterns"))
	viper.SetDefault("plugins_dir", filepath.Join(home, dotBase, "plugins"))
	viper.SetDefault("editor", getDefaultEditor())
	viper.SetDefault("ide", "")
	viper.SetDefault("edit_mode", "auto")
	viper.SetDefault("git_init", true)
	viper.SetDefault("create_readme", true)
	viper.SetDefault("interactive", false)
	viper.SetDefault("default_org", "")
	viper.SetDefault("shell_alias", "")

	// Config file settings
	viper.SetConfigName(configName)
	viper.SetConfigType("yaml")
	viper.AddConfigPath(home)
	viper.AddConfigPath(".")

	// Environment variables
	viper.SetEnvPrefix(envPrefix)
	viper.AutomaticEnv()

	// Read config file (ignore if not found)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("reading config file: %w", err)
		}
	}

	// Unmarshal into struct
	if err := viper.Unmarshal(&C); err != nil {
		return fmt.Errorf("unmarshaling config: %w", err)
	}

	C.Persona = persona
	C.RootDir = expandHome(C.RootDir, home)
	C.PatternsDir = expandHome(C.PatternsDir, home)
	C.PluginsDir = expandHome(C.PluginsDir, home)
	return nil
}

func expandHome(path, home string) string {
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}

// EnvPrefix returns the environment variable prefix for the current persona.
func EnvPrefix() string {
	if C.Persona != "" {
		return "HACK_" + strings.ToUpper(strings.ReplaceAll(C.Persona, "-", "_"))
	}
	return "HACK"
}

// ConfigFilePath returns the path to the config file, or empty if not using one.
func ConfigFilePath() string {
	return viper.ConfigFileUsed()
}

// Source returns how a config key got its current value:
// "file", "env", "flag", or "default".
func Source(key string) string {
	envKey := EnvPrefix() + "_" + strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
	if _, ok := os.LookupEnv(envKey); ok {
		return "env"
	}
	if viper.ConfigFileUsed() != "" && viper.InConfig(key) {
		return "file"
	}
	return "default"
}

// Set sets a config value at runtime.
func Set(key string, value interface{}) {
	viper.Set(key, value)
	_ = viper.Unmarshal(&C)
}

func getDefaultEditor() string {
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor
	}
	return "vim"
}
