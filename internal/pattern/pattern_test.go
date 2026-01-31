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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create a temporary pattern directory
	tmpDir := t.TempDir()
	patternDir := filepath.Join(tmpDir, "test-pattern")
	if err := os.MkdirAll(patternDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create pattern.yaml
	patternYAML := `name: test-pattern
description: A test pattern
variables:
  - name: name
    required: true
  - name: module
    default: github.com/example/{{.name}}
`
	if err := os.WriteFile(filepath.Join(patternDir, "pattern.yaml"), []byte(patternYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Test loading
	p, err := Load(patternDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if p.Name != "test-pattern" {
		t.Errorf("expected name 'test-pattern', got %q", p.Name)
	}
	if p.Description != "A test pattern" {
		t.Errorf("expected description 'A test pattern', got %q", p.Description)
	}
	if len(p.Variables) != 2 {
		t.Errorf("expected 2 variables, got %d", len(p.Variables))
	}
}

func TestLoadWithoutYAML(t *testing.T) {
	// Create a pattern directory without pattern.yaml
	tmpDir := t.TempDir()
	patternDir := filepath.Join(tmpDir, "my-pattern")
	if err := os.MkdirAll(patternDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Test loading - should use directory name as pattern name
	p, err := Load(patternDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if p.Name != "my-pattern" {
		t.Errorf("expected name 'my-pattern', got %q", p.Name)
	}
}

func TestList(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two pattern directories
	for _, name := range []string{"pattern-a", "pattern-b"} {
		patternDir := filepath.Join(tmpDir, name)
		if err := os.MkdirAll(patternDir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	patterns, err := List(tmpDir)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(patterns) != 2 {
		t.Errorf("expected 2 patterns, got %d", len(patterns))
	}
}

func TestApply(t *testing.T) {
	tmpDir := t.TempDir()

	// Create pattern structure
	patternDir := filepath.Join(tmpDir, "patterns", "test-pattern")
	templateDir := filepath.Join(patternDir, "template")
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create pattern.yaml
	patternYAML := `name: test-pattern
description: Test pattern
`
	if err := os.WriteFile(filepath.Join(patternDir, "pattern.yaml"), []byte(patternYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Create template files
	// Non-template file (copied as-is)
	if err := os.WriteFile(filepath.Join(templateDir, "LICENSE"), []byte("Apache 2.0"), 0644); err != nil {
		t.Fatal(err)
	}

	// Template file
	templateContent := `# {{.name}}

Module: {{.module}}
Year: {{.year}}
`
	if err := os.WriteFile(filepath.Join(templateDir, "README.md.tmpl"), []byte(templateContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create cmd subdirectory with template
	cmdDir := filepath.Join(templateDir, "cmd")
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
		t.Fatal(err)
	}
	cmdTemplate := `package cmd

// Version is {{.name}} version
var Version = "dev"
`
	if err := os.WriteFile(filepath.Join(cmdDir, "root.go.tmpl"), []byte(cmdTemplate), 0644); err != nil {
		t.Fatal(err)
	}

	// Apply the pattern
	destDir := filepath.Join(tmpDir, "output")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{
		"name":   "myproject",
		"module": "github.com/example/myproject",
		"year":   "2026",
	}

	patternsDir := filepath.Join(tmpDir, "patterns")
	if err := Apply(patternsDir, "test-pattern", destDir, vars); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Verify LICENSE was copied
	licenseContent, err := os.ReadFile(filepath.Join(destDir, "LICENSE"))
	if err != nil {
		t.Fatalf("LICENSE not found: %v", err)
	}
	if string(licenseContent) != "Apache 2.0" {
		t.Errorf("LICENSE content mismatch")
	}

	// Verify README.md was created (without .tmpl extension)
	readmeContent, err := os.ReadFile(filepath.Join(destDir, "README.md"))
	if err != nil {
		t.Fatalf("README.md not found: %v", err)
	}
	readmeStr := string(readmeContent)
	if !strings.Contains(readmeStr, "# myproject") {
		t.Errorf("README.md missing project name, got: %s", readmeStr)
	}
	if !strings.Contains(readmeStr, "github.com/example/myproject") {
		t.Errorf("README.md missing module, got: %s", readmeStr)
	}

	// Verify cmd/root.go was created
	rootGoContent, err := os.ReadFile(filepath.Join(destDir, "cmd", "root.go"))
	if err != nil {
		t.Fatalf("cmd/root.go not found: %v", err)
	}
	if !strings.Contains(string(rootGoContent), "// Version is myproject version") {
		t.Errorf("cmd/root.go template not expanded correctly")
	}
}

func TestInstall(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source pattern
	srcDir := filepath.Join(tmpDir, "src-pattern")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}

	patternYAML := `name: installable
description: A pattern to install
`
	if err := os.WriteFile(filepath.Join(srcDir, "pattern.yaml"), []byte(patternYAML), 0644); err != nil {
		t.Fatal(err)
	}

	templateDir := filepath.Join(srcDir, "template")
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, "file.txt"), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Install to destination
	destPatternsDir := filepath.Join(tmpDir, "patterns")
	if err := Install(srcDir, destPatternsDir); err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Verify installation
	installedPattern := filepath.Join(destPatternsDir, "installable")
	if _, err := os.Stat(installedPattern); os.IsNotExist(err) {
		t.Error("pattern not installed")
	}

	// Verify pattern.yaml was copied
	if _, err := os.Stat(filepath.Join(installedPattern, "pattern.yaml")); os.IsNotExist(err) {
		t.Error("pattern.yaml not copied")
	}

	// Verify template file was copied
	if _, err := os.Stat(filepath.Join(installedPattern, "template", "file.txt")); os.IsNotExist(err) {
		t.Error("template file not copied")
	}
}
