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

// Package security provides input validation and path safety checks.
package security

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cloudygreybeard/hack/internal/log"
)

// ValidNamePattern matches valid project/app names.
// Allows lowercase letters, numbers, and hyphens, starting with a letter.
var ValidNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// ErrInvalidName is returned when a name contains invalid characters.
type ErrInvalidName struct {
	Name   string
	Reason string
}

func (e ErrInvalidName) Error() string {
	return fmt.Sprintf("invalid name %q: %s", e.Name, e.Reason)
}

// ErrPathTraversal is returned when a path attempts to escape its base.
type ErrPathTraversal struct {
	Path     string
	BasePath string
}

func (e ErrPathTraversal) Error() string {
	return fmt.Sprintf("path traversal detected: %q escapes base %q", e.Path, e.BasePath)
}

// ValidateName checks if a name is safe for use as a project/app name.
// Returns an error if the name contains path traversal sequences,
// control characters, or other potentially dangerous content.
func ValidateName(name string) error {
	if name == "" {
		return ErrInvalidName{Name: name, Reason: "name cannot be empty"}
	}

	// Check for path traversal
	if strings.Contains(name, "..") {
		log.SecurityEvent("path traversal attempt in name", name)
		return ErrInvalidName{Name: name, Reason: "contains path traversal sequence (..)"}
	}

	// Check for path separators
	if strings.ContainsAny(name, "/\\") {
		log.SecurityEvent("path separator in name", name)
		return ErrInvalidName{Name: name, Reason: "contains path separator"}
	}

	// Check for null bytes
	if strings.ContainsRune(name, '\x00') {
		log.SecurityEvent("null byte in name", name)
		return ErrInvalidName{Name: name, Reason: "contains null byte"}
	}

	// Check for control characters
	for _, r := range name {
		if r < 32 || r == 127 {
			log.SecurityEvent("control character in name", name)
			return ErrInvalidName{Name: name, Reason: "contains control characters"}
		}
	}

	// Warn if name doesn't match recommended pattern (but don't reject)
	if !ValidNamePattern.MatchString(name) {
		log.Debug("name %q does not match recommended pattern [a-z][a-z0-9-]*", name)
	}

	return nil
}

// ValidatePathComponent checks a single path component for safety.
func ValidatePathComponent(component string) error {
	if component == "" {
		return ErrInvalidName{Name: component, Reason: "path component cannot be empty"}
	}

	if component == "." || component == ".." {
		return ErrInvalidName{Name: component, Reason: "relative path component not allowed"}
	}

	if strings.ContainsAny(component, "/\\") {
		return ErrInvalidName{Name: component, Reason: "contains path separator"}
	}

	if strings.ContainsRune(component, '\x00') {
		return ErrInvalidName{Name: component, Reason: "contains null byte"}
	}

	return nil
}

// IsPathSafe checks if a target path is safely contained within a base path.
// It resolves both paths to absolute paths and verifies that the target
// is a child of (or equal to) the base path.
func IsPathSafe(basePath, targetPath string) (bool, error) {
	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return false, fmt.Errorf("resolving base path: %w", err)
	}

	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return false, fmt.Errorf("resolving target path: %w", err)
	}

	// Clean paths to normalize any remaining . or ..
	absBase = filepath.Clean(absBase)
	absTarget = filepath.Clean(absTarget)

	// Check if target starts with base (plus separator to avoid prefix matching)
	// e.g., /foo/bar should not match /foo/barbaz
	if absTarget == absBase {
		return true, nil
	}

	baseWithSep := absBase + string(filepath.Separator)
	if strings.HasPrefix(absTarget, baseWithSep) {
		return true, nil
	}

	log.Debug("path safety check failed: %q not in %q", absTarget, absBase)
	return false, nil
}

// EnsurePathSafe is like IsPathSafe but returns an error if the path is unsafe.
func EnsurePathSafe(basePath, targetPath string) error {
	safe, err := IsPathSafe(basePath, targetPath)
	if err != nil {
		return err
	}
	if !safe {
		log.SecurityEvent("path traversal blocked", targetPath, "base="+basePath)
		return ErrPathTraversal{Path: targetPath, BasePath: basePath}
	}
	return nil
}

// SanitizeForPath removes or replaces characters that are problematic in paths.
// This is a lossy operation and should be used for display/logging, not as
// a security control (use ValidateName for validation).
func SanitizeForPath(s string) string {
	// Replace path separators and other problematic chars with hyphen
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		"..", "-",
		"\x00", "",
	)
	result := replacer.Replace(s)

	// Remove any remaining control characters
	cleaned := strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, result)

	return cleaned
}
