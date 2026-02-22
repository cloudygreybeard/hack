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

package pattern

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// InstalledMeta stores metadata about an installed pattern's origin.
type InstalledMeta struct {
	Source  string `yaml:"source,omitempty"`
	Version string `yaml:"version,omitempty"`
}

const installedMetaFile = ".installed.yaml"

// SaveInstalledMeta writes installation metadata alongside the pattern.
func SaveInstalledMeta(patternsDir, patternName string, meta InstalledMeta) error {
	path := filepath.Join(patternsDir, patternName, installedMetaFile)
	data, err := yaml.Marshal(meta)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadInstalledMeta reads installation metadata for a pattern.
func LoadInstalledMeta(patternsDir, patternName string) (InstalledMeta, error) {
	path := filepath.Join(patternsDir, patternName, installedMetaFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return InstalledMeta{}, nil
		}
		return InstalledMeta{}, err
	}
	var m InstalledMeta
	if err := yaml.Unmarshal(data, &m); err != nil {
		return InstalledMeta{}, err
	}
	return m, nil
}

// CompareVersions compares two semver-like version strings.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
// Handles simple major.minor.patch format.
func CompareVersions(a, b string) int {
	if a == b {
		return 0
	}
	if a == "" {
		return -1
	}
	if b == "" {
		return 1
	}

	aParts := parseVersion(a)
	bParts := parseVersion(b)

	for i := 0; i < 3; i++ {
		if aParts[i] < bParts[i] {
			return -1
		}
		if aParts[i] > bParts[i] {
			return 1
		}
	}
	return 0
}

func parseVersion(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	var result [3]int
	for i := 0; i < len(parts) && i < 3; i++ {
		// Strip any pre-release suffix
		num := strings.SplitN(parts[i], "-", 2)[0]
		n, err := strconv.Atoi(num)
		if err == nil {
			result[i] = n
		}
	}
	return result
}

// OutdatedInfo describes a pattern with a newer version available.
type OutdatedInfo struct {
	Name             string
	InstalledVersion string
	AvailableVersion string
	Source           string
}

// CheckOutdated checks a pattern source for a newer version.
// Returns nil if the pattern is up to date or has no version/source info.
func CheckOutdated(patternsDir, patternName string) (*OutdatedInfo, error) {
	installed, err := Load(filepath.Join(patternsDir, patternName))
	if err != nil {
		return nil, err
	}

	meta, err := LoadInstalledMeta(patternsDir, patternName)
	if err != nil {
		return nil, err
	}

	source := installed.Source
	if source == "" {
		source = meta.Source
	}

	if source == "" || installed.Version == "" {
		return nil, nil
	}

	// For local sources, load the pattern from the source path
	if !IsRemoteSource(source) {
		srcPattern, err := Load(source)
		if err != nil {
			return nil, fmt.Errorf("loading source pattern %s: %w", source, err)
		}
		if srcPattern.Version == "" {
			return nil, nil
		}
		if CompareVersions(installed.Version, srcPattern.Version) < 0 {
			return &OutdatedInfo{
				Name:             patternName,
				InstalledVersion: installed.Version,
				AvailableVersion: srcPattern.Version,
				Source:           source,
			}, nil
		}
		return nil, nil
	}

	return nil, nil
}
