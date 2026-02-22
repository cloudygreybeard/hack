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
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cloudygreybeard/hack/internal/log"
)

// IsRemoteSource returns true if the source looks like a URL or Git remote.
func IsRemoteSource(src string) bool {
	if strings.HasPrefix(src, "https://") || strings.HasPrefix(src, "http://") {
		return true
	}
	if strings.HasPrefix(src, "git@") || strings.HasPrefix(src, "ssh://") {
		return true
	}
	// GitHub shorthand: owner/repo or owner/repo//subpath
	if !strings.HasPrefix(src, "/") && !strings.HasPrefix(src, ".") && strings.Count(src, "/") >= 1 {
		parts := strings.SplitN(src, "//", 2)
		base := parts[0]
		if strings.Count(base, "/") == 1 && !strings.Contains(base, " ") {
			return true
		}
	}
	return false
}

// InstallFromRemote fetches a pattern from a remote source and installs it.
// Supports:
//   - Git clone URLs (https://, git@, ssh://)
//   - GitHub shorthand (owner/repo or owner/repo//subpath)
//   - Tarball URLs (.tar.gz, .tgz)
func InstallFromRemote(src, patternsDir string) error {
	tmpDir, err := os.MkdirTemp("", "hack-pattern-remote-*")
	if err != nil {
		return fmt.Errorf("creating temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Determine the source type and fetch
	subpath := ""
	fetchSrc := src

	// Handle GitHub shorthand (owner/repo//subpath)
	if !strings.HasPrefix(src, "https://") && !strings.HasPrefix(src, "http://") &&
		!strings.HasPrefix(src, "git@") && !strings.HasPrefix(src, "ssh://") {
		parts := strings.SplitN(src, "//", 2)
		if len(parts) == 2 {
			subpath = parts[1]
		}
		fetchSrc = "https://github.com/" + parts[0] + ".git"
	}

	if strings.HasSuffix(fetchSrc, ".tar.gz") || strings.HasSuffix(fetchSrc, ".tgz") {
		if err := fetchTarball(fetchSrc, tmpDir); err != nil {
			return err
		}
	} else {
		if err := gitClone(fetchSrc, tmpDir); err != nil {
			return err
		}
	}

	// If subpath specified, install from within the cloned repo
	installDir := tmpDir
	if subpath != "" {
		installDir = filepath.Join(tmpDir, subpath)
		if _, err := os.Stat(installDir); os.IsNotExist(err) {
			return fmt.Errorf("subpath %q not found in remote repository", subpath)
		}
	}

	// Check if the target is a single pattern or a directory of patterns
	if _, err := os.Stat(filepath.Join(installDir, "pattern.yaml")); err == nil {
		log.Verbose("installing single pattern from %s", src)
		return Install(installDir, patternsDir)
	}

	// Try as a directory of patterns (like pattern sync)
	entries, err := os.ReadDir(installDir)
	if err != nil {
		return fmt.Errorf("reading fetched directory: %w", err)
	}

	var installed int
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		patternDir := filepath.Join(installDir, entry.Name())
		if _, err := os.Stat(filepath.Join(patternDir, "pattern.yaml")); os.IsNotExist(err) {
			continue
		}
		if err := Install(patternDir, patternsDir); err != nil {
			log.Warn("failed to install %s: %v", entry.Name(), err)
			continue
		}
		log.Info("installed: %s", entry.Name())
		installed++
	}

	if installed == 0 {
		return fmt.Errorf("no valid patterns found at %s", src)
	}

	log.Info("%d pattern(s) installed from remote", installed)
	return nil
}

func gitClone(url, destDir string) error {
	log.Verbose("cloning %s", url)

	// Shallow clone for speed
	cmd := exec.Command("git", "clone", "--depth", "1", url, destDir)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}
	return nil
}

func fetchTarball(url, destDir string) error {
	log.Verbose("downloading %s", url)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("downloading tarball: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("decompressing: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tarball: %w", err)
		}

		// Strip the first path component (common in GitHub archives)
		parts := strings.SplitN(hdr.Name, "/", 2)
		if len(parts) < 2 || parts[1] == "" {
			continue
		}
		target := filepath.Join(destDir, parts[1])

		// Security: ensure path stays within destDir
		absTarget, _ := filepath.Abs(target)
		absDest, _ := filepath.Abs(destDir)
		if !strings.HasPrefix(absTarget, absDest+string(os.PathSeparator)) && absTarget != absDest {
			continue
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}
	return nil
}
