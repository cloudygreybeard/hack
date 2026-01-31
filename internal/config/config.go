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

	"github.com/spf13/viper"
)

// Config holds the application configuration.
type Config struct {
	RootDir      string `mapstructure:"root_dir"`
	PatternsDir  string `mapstructure:"patterns_dir"`
	Editor       string `mapstructure:"editor"`
	GitInit      bool   `mapstructure:"git_init"`
	CreateReadme bool   `mapstructure:"create_readme"`
	Interactive  bool   `mapstructure:"interactive"`
	DefaultOrg   string `mapstructure:"default_org"`
}

// C is the global configuration instance.
var C Config

// Init initializes the configuration from file, environment, and defaults.
func Init() error {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}

	// Set defaults
	viper.SetDefault("root_dir", filepath.Join(home, "hack"))
	viper.SetDefault("patterns_dir", filepath.Join(home, ".hack", "patterns"))
	viper.SetDefault("editor", getDefaultEditor())
	viper.SetDefault("git_init", true)
	viper.SetDefault("create_readme", true)
	viper.SetDefault("interactive", false)
	viper.SetDefault("default_org", "")

	// Config file settings
	viper.SetConfigName(".hack")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(home)
	viper.AddConfigPath(".")

	// Environment variables
	viper.SetEnvPrefix("HACK")
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

	return nil
}

// ConfigFilePath returns the path to the config file, or empty if not using one.
func ConfigFilePath() string {
	return viper.ConfigFileUsed()
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
