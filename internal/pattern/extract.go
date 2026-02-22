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
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudygreybeard/hack/internal/log"
	"github.com/cloudygreybeard/hack/internal/workspace"
	"gopkg.in/yaml.v3"
)

// ExtractOptions configures pattern extraction from a workspace.
type ExtractOptions struct {
	PatternName  string
	OutputDir    string
	AppOnly      string   // extract only this app subdirectory
	Templatise   bool
	ExtraExclude []string // additional globs to exclude
}

// ExtractVars holds the concrete values to reverse-substitute.
type ExtractVars struct {
	Name    string
	AppName string
	Module  string
	Year    string
	Org     string
	Title   string // TitleCase of AppName
}

var defaultExcludeDirs = map[string]bool{
	".git":         true,
	"bin":          true,
	"vendor":       true,
	"node_modules": true,
}

var defaultExcludeFiles = map[string]bool{
	".hack.yaml":     true,
	".installed.yaml": true,
}

// Extract creates a pattern from an existing workspace directory.
func Extract(srcDir string, opts ExtractOptions) error {
	vars, err := inferVars(srcDir, opts)
	if err != nil {
		return fmt.Errorf("inferring variables: %w", err)
	}

	log.Verbose("extracted variables: name=%s app_name=%s module=%s year=%s org=%s",
		vars.Name, vars.AppName, vars.Module, vars.Year, vars.Org)

	outDir := opts.OutputDir
	if outDir == "" {
		outDir = opts.PatternName
	}

	templateDir := filepath.Join(outDir, "template")
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Determine the root to walk
	walkRoot := srcDir
	if opts.AppOnly != "" {
		walkRoot = filepath.Join(srcDir, opts.AppOnly)
		if _, err := os.Stat(walkRoot); os.IsNotExist(err) {
			return fmt.Errorf("app directory %q not found in workspace", opts.AppOnly)
		}
	}

	// Build replacement pairs (longest first to avoid partial matches)
	replacements := buildReplacements(vars, opts.Templatise)

	var filesExtracted int

	walkErr := filepath.WalkDir(walkRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(walkRoot, path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		// Check exclusions
		if d.IsDir() {
			baseName := filepath.Base(relPath)
			if defaultExcludeDirs[baseName] || strings.HasPrefix(baseName, ".") {
				return filepath.SkipDir
			}
			for _, excl := range opts.ExtraExclude {
				if matched, _ := filepath.Match(excl, baseName); matched {
					return filepath.SkipDir
				}
			}
			return nil
		}

		baseName := filepath.Base(relPath)
		if defaultExcludeFiles[baseName] {
			return nil
		}
		for _, excl := range opts.ExtraExclude {
			if matched, _ := filepath.Match(excl, baseName); matched {
				return nil
			}
		}

		// Build destination path, wrapping the app dir in {{app_name}} if at workspace level
		destRelPath := relPath
		if opts.AppOnly == "" && opts.Templatise && vars.AppName != "" {
			destRelPath = replacePathComponent(destRelPath, vars.AppName, "{{app_name}}")
		}

		// Read source file
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", relPath, err)
		}

		// Check if binary
		if isBinary(content) {
			destPath := filepath.Join(templateDir, destRelPath)
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return err
			}
			log.FileCreated(destRelPath)
			filesExtracted++
			return os.WriteFile(destPath, content, 0644)
		}

		// Apply templatisation
		result := string(content)
		substituted := false
		if opts.Templatise {
			// Escape pre-existing Go template expressions so they survive
			// round-trip through text/template (e.g., goreleaser's {{ .Version }})
			if strings.Contains(result, "{{") {
				result = escapeExistingTemplateExprs(result)
				substituted = true
			}

			for _, r := range replacements {
				if strings.Contains(result, r.from) {
					result = strings.ReplaceAll(result, r.from, r.to)
					substituted = true
				}
			}
		}

		// Add .tmpl suffix if content was templatised
		if substituted {
			destRelPath += ".tmpl"
		}

		destPath := filepath.Join(templateDir, destRelPath)
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		log.FileCreated(destRelPath)
		filesExtracted++
		return os.WriteFile(destPath, []byte(result), 0644)
	})

	if walkErr != nil {
		return walkErr
	}

	// Generate pattern.yaml
	if err := generatePatternYAML(outDir, opts, vars); err != nil {
		return fmt.Errorf("generating pattern.yaml: %w", err)
	}

	log.Verbose("extracted %d files to %s", filesExtracted, outDir)
	return nil
}

type replacement struct {
	from string
	to   string
}

func buildReplacements(vars ExtractVars, templatise bool) []replacement {
	if !templatise {
		return nil
	}

	var reps []replacement

	// Order matters: longer/more-specific strings first to avoid partial matches
	if vars.Module != "" {
		reps = append(reps, replacement{vars.Module, "{{.module}}"})
	}
	if vars.Title != "" && vars.Title != vars.AppName {
		reps = append(reps, replacement{vars.Title, "{{.Name}}"})
	}
	if vars.Org != "" {
		reps = append(reps, replacement{vars.Org, "{{.org}}"})
	}
	if vars.AppName != "" {
		reps = append(reps, replacement{vars.AppName, "{{.app_name}}"})
	}
	if vars.Name != "" && vars.Name != vars.AppName {
		reps = append(reps, replacement{vars.Name, "{{.name}}"})
	}
	if vars.Year != "" {
		reps = append(reps, replacement{vars.Year, "{{.year}}"})
	}

	return reps
}

func replacePathComponent(path, old, replacement string) string {
	parts := strings.Split(path, string(os.PathSeparator))
	for i, part := range parts {
		if part == old {
			parts[i] = replacement
		}
	}
	return filepath.Join(parts...)
}

// escapeExistingTemplateExprs wraps pre-existing {{ }} expressions in the
// source file with Go template literal escaping, so they survive processing
// by text/template during pattern application. Our own injected variables
// (like {{.name}}) are added after this step and are not escaped.
func escapeExistingTemplateExprs(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	for i := 0; i < len(s); i++ {
		if i+1 < len(s) && s[i] == '{' && s[i+1] == '{' {
			// Find the matching }}
			end := strings.Index(s[i+2:], "}}")
			if end >= 0 {
				inner := s[i+2 : i+2+end]
				b.WriteString(`{{"` + "{{" + `"}}`)
				b.WriteString(inner)
				b.WriteString(`{{"` + "}}" + `"}}`)
				i = i + 2 + end + 1 // skip past }}
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func isBinary(data []byte) bool {
	check := data
	if len(check) > 512 {
		check = check[:512]
	}
	for _, b := range check {
		if b == 0 {
			return true
		}
	}
	return false
}

func inferVars(srcDir string, opts ExtractOptions) (ExtractVars, error) {
	var vars ExtractVars

	// Try to load workspace metadata
	meta, _ := workspace.Load(srcDir)
	if meta.MetadataObj.Name != "" {
		vars.Name = meta.MetadataObj.Name
	} else {
		vars.Name = filepath.Base(srcDir)
		// Strip date prefix if present (e.g., "2026-01-26.my-project" -> "my-project")
		if parts := strings.SplitN(vars.Name, ".", 2); len(parts) == 2 {
			vars.Name = parts[1]
		}
	}

	// Infer year from directory name date prefix
	dirBase := filepath.Base(srcDir)
	if len(dirBase) >= 4 {
		vars.Year = dirBase[:4]
	}

	// Determine app name
	if opts.AppOnly != "" {
		vars.AppName = opts.AppOnly
	} else {
		vars.AppName = inferAppDir(srcDir, vars.Name)
	}

	// Infer module from go.mod
	modPath := ""
	if vars.AppName != "" {
		modPath = filepath.Join(srcDir, vars.AppName, "go.mod")
	}
	if modPath == "" || !fileExists(modPath) {
		modPath = filepath.Join(srcDir, "go.mod")
	}
	if fileExists(modPath) {
		vars.Module = readModulePath(modPath)
	}

	// Infer org from module path
	if vars.Module != "" && vars.Org == "" {
		parts := strings.Split(vars.Module, "/")
		if len(parts) >= 2 {
			vars.Org = parts[1]
		}
	}

	// TitleCase
	vars.Title = toExtractTitle(vars.AppName)

	// Override pattern name
	if opts.PatternName == "" {
		opts.PatternName = vars.AppName
		if opts.PatternName == "" {
			opts.PatternName = vars.Name
		}
	}

	return vars, nil
}

func inferAppDir(srcDir, name string) string {
	// First, try the workspace name
	if dirHasGoCode(filepath.Join(srcDir, name)) {
		return name
	}

	// Scan for first subdirectory with go.mod or main.go
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") || defaultExcludeDirs[e.Name()] {
			continue
		}
		if dirHasGoCode(filepath.Join(srcDir, e.Name())) {
			return e.Name()
		}
	}
	return ""
}

func dirHasGoCode(dir string) bool {
	if !fileExists(dir) {
		return false
	}
	return fileExists(filepath.Join(dir, "go.mod")) || fileExists(filepath.Join(dir, "main.go"))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func readModulePath(goModPath string) string {
	f, err := os.Open(goModPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

func toExtractTitle(s string) string {
	if s == "" {
		return ""
	}
	parts := strings.Split(s, "-")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

func generatePatternYAML(outDir string, opts ExtractOptions, vars ExtractVars) error {
	name := opts.PatternName
	if name == "" {
		name = vars.AppName
		if name == "" {
			name = vars.Name
		}
	}

	p := Pattern{
		Name:        name,
		Description: fmt.Sprintf("Pattern extracted from %s", vars.Name),
		Version:     "0.1.0",
		Variables: []Variable{
			{Name: "name", Description: "Workspace/project name", Required: true},
			{Name: "app_name", Description: "App directory name (defaults to name)", Default: ""},
			{Name: "module", Description: "Go module path", Default: ""},
			{Name: "year", Description: "Copyright year", Default: vars.Year},
			{Name: "date", Description: "Creation date (YYYY-MM-DD)"},
		},
	}

	if vars.Org != "" {
		p.Variables = append(p.Variables,
			Variable{Name: "org", Description: "GitHub org or copyright holder", Default: vars.Org})
	}

	data, err := yaml.Marshal(p)
	if err != nil {
		return err
	}

	path := filepath.Join(outDir, "pattern.yaml")
	log.FileCreated("pattern.yaml")
	return os.WriteFile(path, data, 0644)
}
