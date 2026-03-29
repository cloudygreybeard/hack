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

// Package cursor provides Cursor IDE workspace storage migration.
package cursor

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/cloudygreybeard/hack/internal/log"
)

// storageBasePath returns the platform-specific Cursor workspace storage directory.
func storageBasePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Cursor", "User"), nil
	case "linux":
		return filepath.Join(home, ".config", "Cursor", "User"), nil
	default:
		return "", fmt.Errorf("unsupported platform %q for Cursor migration", runtime.GOOS)
	}
}

// workspaceHash computes the Cursor workspace storage hash for a directory.
// The salt is platform-specific (birthtime on macOS, inode on Linux) and
// provided by platformSalt in the _darwin.go / _linux.go files.
func workspaceHash(fsPath string) (string, error) {
	salt, err := platformSalt(fsPath)
	if err != nil {
		return "", err
	}
	raw := fsPath + salt
	hash := md5.Sum([]byte(raw))
	return hex.EncodeToString(hash[:]), nil
}

type workspaceJSON struct {
	Folder    string `json:"folder,omitempty"`
	Workspace string `json:"workspace,omitempty"`
}

// findStorageDir locates the workspace storage directory for a given path
// by scanning workspace.json files. This is more reliable than computing
// the hash when the directory may have been moved/recreated.
func findStorageDir(userDir, fsPath string) (string, error) {
	storageDir := filepath.Join(userDir, "workspaceStorage")
	uri := "file:///" + strings.TrimPrefix(fsPath, "/")

	entries, err := os.ReadDir(storageDir)
	if err != nil {
		return "", fmt.Errorf("reading workspace storage: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		wjPath := filepath.Join(storageDir, entry.Name(), "workspace.json")
		data, err := os.ReadFile(wjPath)
		if err != nil {
			continue
		}
		var wj workspaceJSON
		if err := json.Unmarshal(data, &wj); err != nil {
			continue
		}
		if wj.Folder == uri {
			return filepath.Join(storageDir, entry.Name()), nil
		}
	}
	return "", nil
}

// copyDir recursively copies a directory tree.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}

// updateWorkspaceJSON rewrites workspace.json with the new folder URI.
func updateWorkspaceJSON(storagePath, newFsPath string) error {
	wjPath := filepath.Join(storagePath, "workspace.json")
	newURI := "file:///" + strings.TrimPrefix(newFsPath, "/")

	wj := workspaceJSON{Folder: newURI}
	data, err := json.MarshalIndent(wj, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling workspace.json: %w", err)
	}
	data = append(data, '\n')
	return os.WriteFile(wjPath, data, 0644)
}

// updateStateDB replaces old path references in state.vscdb using the sqlite3 CLI.
func updateStateDB(storagePath, oldPath, newPath string) error {
	dbPath := filepath.Join(storagePath, "state.vscdb")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		log.Debug("no state.vscdb found, skipping")
		return nil
	}

	oldURI := "file:///" + strings.TrimPrefix(oldPath, "/")
	newURI := "file:///" + strings.TrimPrefix(newPath, "/")

	replacements := [][2]string{
		{oldPath, newPath},
		{oldURI, newURI},
	}

	for _, r := range replacements {
		query := fmt.Sprintf(
			`UPDATE ItemTable SET value = REPLACE(value, '%s', '%s') WHERE value LIKE '%%%s%%';`,
			escapeSQLite(r[0]), escapeSQLite(r[1]), escapeSQLite(r[0]),
		)

		cmd := exec.Command("sqlite3", dbPath, query)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("updating state.vscdb: %w: %s", err, output)
		}
	}

	log.Debug("updated path references in state.vscdb")
	return nil
}

func escapeSQLite(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// updateGlobalStorage updates references in globalStorage/storage.json.
func updateGlobalStorage(userDir, oldPath, newPath string) error {
	storagePath := filepath.Join(userDir, "globalStorage", "storage.json")
	data, err := os.ReadFile(storagePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Debug("no global storage.json found, skipping")
			return nil
		}
		return fmt.Errorf("reading storage.json: %w", err)
	}

	oldURI := "file:///" + strings.TrimPrefix(oldPath, "/")
	newURI := "file:///" + strings.TrimPrefix(newPath, "/")

	content := string(data)
	updated := strings.ReplaceAll(content, oldURI, newURI)
	updated = strings.ReplaceAll(updated, oldPath, newPath)

	if updated == content {
		log.Debug("no references to old path in storage.json")
		return nil
	}

	if err := os.WriteFile(storagePath, []byte(updated), 0644); err != nil {
		return fmt.Errorf("writing storage.json: %w", err)
	}
	log.Debug("updated global storage.json")
	return nil
}

// CheckWorkspace queries Cursor's status to determine whether Cursor is
// running and whether a specific workspace is open. Uses "cursor --status"
// which reports all open windows and their folders.
func CheckWorkspace(fsPath string) (running bool, workspaceOpen bool) {
	cmd := exec.Command("cursor", "--status")
	output, err := cmd.Output()
	if err != nil {
		log.Debug("cursor --status: %v (Cursor likely not running)", err)
		return false, false
	}

	dirName := filepath.Base(fsPath)
	for _, line := range strings.Split(string(output), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "| ") && strings.Contains(trimmed, "Folder ("+dirName+")") {
			return true, true
		}
		if strings.HasPrefix(trimmed, "| ") && strings.Contains(trimmed, "Window (") && strings.Contains(trimmed, dirName+")") {
			return true, true
		}
	}
	return true, false
}

// MigrateWorkspace migrates Cursor workspace storage from oldPath to newPath.
// The old storage directory is preserved as a backup.
// Call this BEFORE renaming the project directory (oldPath must still exist).
func MigrateWorkspace(oldPath, newPath string) error {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		log.Warn("Cursor migration not supported on %s, skipping", runtime.GOOS)
		return nil
	}

	userDir, err := storageBasePath()
	if err != nil {
		return err
	}

	oldStorageDir, err := findStorageDir(userDir, oldPath)
	if err != nil {
		return fmt.Errorf("finding old workspace storage: %w", err)
	}
	if oldStorageDir == "" {
		log.Verbose("no Cursor workspace storage found for %s", oldPath)
		return nil
	}
	log.Debug("found old workspace storage: %s", filepath.Base(oldStorageDir))

	newHash, err := workspaceHash(newPath)
	if err != nil {
		return fmt.Errorf("computing new workspace hash: %w", err)
	}
	log.Debug("new workspace hash: %s", newHash)

	newStorageDir := filepath.Join(userDir, "workspaceStorage", newHash)

	if _, err := os.Stat(newStorageDir); err == nil {
		log.Verbose("workspace storage %s already exists, skipping copy", newHash[:12])
	} else {
		log.Verbose("copying workspace storage to %s", newHash[:12])
		if err := copyDir(oldStorageDir, newStorageDir); err != nil {
			return fmt.Errorf("copying workspace storage: %w", err)
		}
	}

	if err := updateWorkspaceJSON(newStorageDir, newPath); err != nil {
		return fmt.Errorf("updating workspace.json: %w", err)
	}

	if err := updateStateDB(newStorageDir, oldPath, newPath); err != nil {
		log.Warn("updating state.vscdb: %v", err)
	}

	if err := updateGlobalStorage(userDir, oldPath, newPath); err != nil {
		log.Warn("updating global storage: %v", err)
	}

	log.Verbose("Cursor workspace storage migrated")
	return nil
}
