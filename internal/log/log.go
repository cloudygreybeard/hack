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

// Package log provides structured logging with configurable verbosity levels.
package log

import (
	"fmt"
	"os"
	"strings"
)

// Level represents log verbosity levels.
type Level int

const (
	// LevelQuiet suppresses all informational output.
	LevelQuiet Level = -1
	// LevelNormal is the default level (errors and essential info).
	LevelNormal Level = 0
	// LevelVerbose shows additional operational details.
	LevelVerbose Level = 1
	// LevelDebug shows detailed debugging information.
	LevelDebug Level = 2
)

var currentLevel = LevelNormal

// SetLevel sets the global log verbosity level.
func SetLevel(level Level) {
	currentLevel = level
}

// GetLevel returns the current log verbosity level.
func GetLevel() Level {
	return currentLevel
}

// SetVerbosity sets verbosity from a count (e.g., -v, -vv, -vvv).
func SetVerbosity(count int) {
	if count < 0 {
		currentLevel = LevelQuiet
	} else if count > int(LevelDebug) {
		currentLevel = LevelDebug
	} else {
		currentLevel = Level(count)
	}
}

// Error prints an error message to stderr (always shown unless quiet).
func Error(format string, args ...interface{}) {
	if currentLevel >= LevelQuiet {
		fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	}
}

// Warn prints a warning message to stderr.
func Warn(format string, args ...interface{}) {
	if currentLevel >= LevelNormal {
		fmt.Fprintf(os.Stderr, "warning: "+format+"\n", args...)
	}
}

// Info prints an informational message to stderr.
func Info(format string, args ...interface{}) {
	if currentLevel >= LevelNormal {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
}

// Verbose prints a message at verbose level to stderr.
func Verbose(format string, args ...interface{}) {
	if currentLevel >= LevelVerbose {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
}

// Debug prints a message at debug level to stderr.
func Debug(format string, args ...interface{}) {
	if currentLevel >= LevelDebug {
		fmt.Fprintf(os.Stderr, "[debug] "+format+"\n", args...)
	}
}

// FileCreated logs a file creation event.
func FileCreated(path string) {
	if currentLevel >= LevelVerbose {
		fmt.Fprintf(os.Stderr, "  create: %s\n", path)
	}
}

// FileSkipped logs a skipped file event.
func FileSkipped(path, reason string) {
	if currentLevel >= LevelVerbose {
		fmt.Fprintf(os.Stderr, "  skip: %s (%s)\n", path, reason)
	}
}

// DirCreated logs a directory creation event.
func DirCreated(path string) {
	if currentLevel >= LevelDebug {
		fmt.Fprintf(os.Stderr, "  mkdir: %s\n", path)
	}
}

// TemplateProcessed logs template processing.
func TemplateProcessed(src, dest string) {
	if currentLevel >= LevelDebug {
		fmt.Fprintf(os.Stderr, "  template: %s -> %s\n", src, dest)
	}
}

// SecurityEvent logs a security-relevant event.
func SecurityEvent(event string, details ...string) {
	if currentLevel >= LevelNormal {
		msg := "security: " + event
		if len(details) > 0 {
			msg += " (" + strings.Join(details, ", ") + ")"
		}
		fmt.Fprintf(os.Stderr, "%s\n", msg)
	}
}
