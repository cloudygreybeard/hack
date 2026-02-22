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

// Package workspace provides metadata management for hack workspaces.
package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const metadataFile = ".hack.yaml"

// Metadata represents workspace metadata stored in .hack.yaml.
type Metadata struct {
	APIVersion  string            `yaml:"apiVersion"`
	Kind        string            `yaml:"kind"`
	MetadataObj MetadataFields    `yaml:"metadata"`
}

// MetadataFields holds the name, labels, and annotations.
type MetadataFields struct {
	Name        string            `yaml:"name"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

// Load reads workspace metadata from a directory's .hack.yaml file.
// Returns empty metadata (not an error) if the file does not exist.
func Load(dir string) (Metadata, error) {
	path := filepath.Join(dir, metadataFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Metadata{}, nil
		}
		return Metadata{}, fmt.Errorf("reading %s: %w", metadataFile, err)
	}

	var m Metadata
	if err := yaml.Unmarshal(data, &m); err != nil {
		return Metadata{}, fmt.Errorf("parsing %s: %w", metadataFile, err)
	}
	return m, nil
}

// Save writes workspace metadata to a directory's .hack.yaml file.
func Save(dir string, m Metadata) error {
	if m.APIVersion == "" {
		m.APIVersion = "hack/v1"
	}
	if m.Kind == "" {
		m.Kind = "Workspace"
	}

	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshaling metadata: %w", err)
	}

	path := filepath.Join(dir, metadataFile)
	return os.WriteFile(path, data, 0644)
}

// HasMetadata returns true if a .hack.yaml exists in the directory.
func HasMetadata(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, metadataFile))
	return err == nil
}

// MatchesSelector returns true if the metadata's labels match all
// key-value pairs in the selector (AND semantics, like k8s matchLabels).
func MatchesSelector(m Metadata, selector map[string]string) bool {
	if len(selector) == 0 {
		return true
	}
	labels := m.MetadataObj.Labels
	if labels == nil {
		return false
	}
	for k, v := range selector {
		if labels[k] != v {
			return false
		}
	}
	return true
}

// ParseSelector parses a comma-separated label selector string into a map.
// Format: "key1=value1,key2=value2"
func ParseSelector(s string) (map[string]string, error) {
	if s == "" {
		return nil, nil
	}
	result := make(map[string]string)
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid selector %q: expected key=value", part)
		}
		result[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
	}
	return result, nil
}

// ParseLabelArg parses a label argument which is either "key=value" (set)
// or "key-" (remove). Returns key, value, and whether it's a removal.
func ParseLabelArg(arg string) (key, value string, remove bool, err error) {
	if strings.HasSuffix(arg, "-") {
		key = strings.TrimSuffix(arg, "-")
		if key == "" {
			return "", "", false, fmt.Errorf("invalid label removal: empty key")
		}
		return key, "", true, nil
	}
	kv := strings.SplitN(arg, "=", 2)
	if len(kv) != 2 {
		return "", "", false, fmt.Errorf("invalid label %q: expected key=value or key-", arg)
	}
	return kv[0], kv[1], false, nil
}

// FormatLabels returns labels as a comma-separated "key=value" string.
func FormatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	parts := make([]string, 0, len(labels))
	for k, v := range labels {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, ",")
}
