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

package cursor

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"syscall"
)

// platformSalt returns the hash salt for macOS: round(birthtimeMs).
func platformSalt(fsPath string) (string, error) {
	info, err := os.Stat(fsPath)
	if err != nil {
		return "", fmt.Errorf("stat %s: %w", fsPath, err)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return "", fmt.Errorf("cannot get syscall.Stat_t for %s", fsPath)
	}
	btimeFloat := float64(stat.Birthtimespec.Sec)*1000.0 + float64(stat.Birthtimespec.Nsec)/1e6
	btimeMs := int64(math.Round(btimeFloat))
	return strconv.FormatInt(btimeMs, 10), nil
}
