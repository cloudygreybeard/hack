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

// Package pattern provides template-based project scaffolding.
package pattern

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/cloudygreybeard/hack/internal/log"
	"github.com/cloudygreybeard/hack/internal/security"
	"gopkg.in/yaml.v3"
)

// Pattern represents a project pattern definition.
type Pattern struct {
	Name        string     `yaml:"name"`
	Description string     `yaml:"description"`
	Variables   []Variable `yaml:"variables"`
	PostCreate  []string   `yaml:"post_create"`
}

// Variable defines a template variable.
type Variable struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
	Default     string `yaml:"default"`
}

// List returns all patterns in the patterns directory.
func List(patternsDir string) ([]Pattern, error) {
	entries, err := os.ReadDir(patternsDir)
	if err != nil {
		return nil, err
	}

	var patterns []Pattern
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		p, err := Load(filepath.Join(patternsDir, entry.Name()))
		if err != nil {
			// Skip invalid patterns
			continue
		}
		patterns = append(patterns, p)
	}

	return patterns, nil
}

// Load reads a pattern from a directory.
func Load(patternPath string) (Pattern, error) {
	metaPath := filepath.Join(patternPath, "pattern.yaml")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		// If no pattern.yaml, use directory name
		return Pattern{
			Name: filepath.Base(patternPath),
		}, nil
	}

	var p Pattern
	if err := yaml.Unmarshal(data, &p); err != nil {
		return Pattern{}, fmt.Errorf("invalid pattern.yaml: %w", err)
	}

	if p.Name == "" {
		p.Name = filepath.Base(patternPath)
	}

	return p, nil
}

// ApplyOptions configures pattern application behavior.
type ApplyOptions struct {
	SkipExisting bool // Don't overwrite existing files
}

// Apply copies a pattern to the destination directory, processing templates.
func Apply(patternsDir, patternName, destDir string, vars map[string]string) error {
	return ApplyWithOptions(patternsDir, patternName, destDir, vars, ApplyOptions{})
}

// ApplyWithOptions copies a pattern with configurable options.
func ApplyWithOptions(patternsDir, patternName, destDir string, vars map[string]string, opts ApplyOptions) error {
	patternPath := filepath.Join(patternsDir, patternName)
	templateDir := filepath.Join(patternPath, "template")

	log.Verbose("applying pattern %q to %s", patternName, destDir)
	log.Debug("pattern path: %s", patternPath)
	log.Debug("template dir: %s", templateDir)

	// Check if pattern exists
	if _, err := os.Stat(patternPath); os.IsNotExist(err) {
		return fmt.Errorf("pattern %q not found in %s", patternName, patternsDir)
	}

	// Check for template directory
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		return fmt.Errorf("pattern %q has no template directory", patternName)
	}

	// Resolve destination directory to absolute path for security checks
	absDestDir, err := filepath.Abs(destDir)
	if err != nil {
		return fmt.Errorf("resolving destination path: %w", err)
	}

	var filesCreated, filesSkipped int

	// Walk the template directory and copy files
	walkErr := filepath.WalkDir(templateDir, func(srcPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path from template directory
		relPath, err := filepath.Rel(templateDir, srcPath)
		if err != nil {
			return err
		}

		// Skip root
		if relPath == "." {
			return nil
		}

		// Process path templates (e.g., {{name}} in directory names)
		destRelPath, err := processPathTemplate(relPath, vars)
		if err != nil {
			return fmt.Errorf("processing path %q: %w", relPath, err)
		}

		destPath := filepath.Join(absDestDir, destRelPath)

		// Security: Validate the destination path is within the target directory
		if err := security.EnsurePathSafe(absDestDir, destPath); err != nil {
			return fmt.Errorf("path safety check failed for %q: %w", destRelPath, err)
		}

		if d.IsDir() {
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return err
			}
			log.DirCreated(destRelPath)
			return nil
		}

		// Skip existing files if requested
		if opts.SkipExisting {
			if _, err := os.Stat(destPath); err == nil {
				log.FileSkipped(destRelPath, "already exists")
				filesSkipped++
				return nil
			}
		}

		if err := copyFile(srcPath, destPath, vars); err != nil {
			return err
		}
		log.FileCreated(destRelPath)
		filesCreated++
		return nil
	})

	if walkErr != nil {
		return walkErr
	}

	log.Verbose("pattern applied: %d files created, %d skipped", filesCreated, filesSkipped)
	return nil
}

// RunPostCreate executes post-create hooks defined in the pattern.
// Each hook is run as a shell command in the destination directory.
func RunPostCreate(p Pattern, destDir string, vars map[string]string) error {
	if len(p.PostCreate) == 0 {
		return nil
	}

	log.Verbose("running %d post-create hook(s)", len(p.PostCreate))
	for i, hook := range p.PostCreate {
		// Expand template variables in the hook command
		expanded, err := expandHookTemplate(hook, vars)
		if err != nil {
			return fmt.Errorf("expanding hook %d: %w", i+1, err)
		}

		log.Debug("post-create hook %d: %s", i+1, expanded)

		cmd := exec.Command("sh", "-c", expanded)
		cmd.Dir = destDir
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("post-create hook %d failed: %w", i+1, err)
		}
		log.Verbose("post-create hook %d completed", i+1)
	}
	return nil
}

// expandHookTemplate processes Go template expressions in a hook command.
func expandHookTemplate(hook string, vars map[string]string) (string, error) {
	if !strings.Contains(hook, "{{") {
		return hook, nil
	}

	tmpl, err := template.New("hook").Parse(hook)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// Install copies a pattern from srcPath to the patterns directory.
func Install(srcPath, patternsDir string) error {
	// Validate source
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return fmt.Errorf("source path %q does not exist", srcPath)
	}

	// Load pattern to get name
	p, err := Load(srcPath)
	if err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}

	destPath := filepath.Join(patternsDir, p.Name)

	// Create patterns directory if needed
	if err := os.MkdirAll(patternsDir, 0755); err != nil {
		return fmt.Errorf("creating patterns directory: %w", err)
	}

	// Remove existing pattern if present
	if err := os.RemoveAll(destPath); err != nil {
		return fmt.Errorf("removing existing pattern: %w", err)
	}

	// Copy the pattern
	return copyDir(srcPath, destPath)
}

// processPathTemplate expands template expressions in file/directory paths.
func processPathTemplate(path string, vars map[string]string) (string, error) {
	// Simple replacement for common patterns
	result := path

	// Strip .tmpl extension
	result = strings.TrimSuffix(result, ".tmpl")

	// Replace {{var}} patterns
	for k, v := range vars {
		result = strings.ReplaceAll(result, "{{"+k+"}}", v)
		result = strings.ReplaceAll(result, "{{ "+k+" }}", v)
		result = strings.ReplaceAll(result, "{{."+k+"}}", v)
		result = strings.ReplaceAll(result, "{{ ."+k+" }}", v)
	}

	return result, nil
}

// copyFile copies a file, processing it as a template if it ends in .tmpl.
func copyFile(src, dest string, vars map[string]string) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// If .tmpl file, process as template
	if strings.HasSuffix(src, ".tmpl") {
		return processTemplate(src, dest, vars, srcInfo.Mode())
	}

	// Otherwise, copy directly
	return copyFileRaw(src, dest, srcInfo.Mode())
}

// processTemplate reads a template file, processes it, and writes the result.
func processTemplate(src, dest string, vars map[string]string, mode fs.FileMode) error {
	log.TemplateProcessed(filepath.Base(src), filepath.Base(dest))

	content, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	tmpl, err := template.New(filepath.Base(src)).Parse(string(content))
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	return os.WriteFile(dest, buf.Bytes(), mode)
}

// copyFileRaw copies a file without template processing.
func copyFileRaw(src, dest string, mode fs.FileMode) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	return err
}

// copyDir recursively copies a directory.
func copyDir(src, dest string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(dest, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		return copyFileRaw(path, destPath, info.Mode())
	})
}
