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

// Package prompt provides interactive prompts for user input.
package prompt

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/cloudygreybeard/hack/internal/pattern"
)

// PatternVariables prompts the user for pattern variables interactively.
func PatternVariables(p pattern.Pattern, defaults map[string]string) (map[string]string, error) {
	// Check if we have a TTY
	if !isInteractive() {
		return defaults, nil
	}

	result := make(map[string]string)
	for k, v := range defaults {
		result[k] = v
	}

	// Collect variables that need prompting
	type varInput struct {
		name     string
		desc     string
		required bool
		defVal   string
		value    string
	}

	var inputs []varInput

	for _, v := range p.Variables {
		// Skip if already has a non-empty value
		if val, ok := result[v.Name]; ok && val != "" {
			continue
		}

		// Determine default value
		defaultVal := v.Default
		if val, ok := defaults[v.Name]; ok && val != "" {
			defaultVal = val
		}

		desc := v.Description
		if desc == "" {
			desc = v.Name
		}

		inputs = append(inputs, varInput{
			name:     v.Name,
			desc:     desc,
			required: v.Required,
			defVal:   defaultVal,
			value:    defaultVal,
		})
	}

	if len(inputs) == 0 {
		return result, nil
	}

	// Build form fields
	var fields []huh.Field
	for i := range inputs {
		input := huh.NewInput().
			Title(inputs[i].desc).
			Value(&inputs[i].value).
			Placeholder(inputs[i].defVal)

		if inputs[i].required {
			idx := i // capture for closure
			input = input.Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("%s is required", inputs[idx].name)
				}
				return nil
			})
		}

		fields = append(fields, input)
	}

	form := huh.NewForm(
		huh.NewGroup(fields...),
	).WithTheme(huh.ThemeBase())

	err := form.Run()
	if err != nil {
		return nil, err
	}

	// Copy values back to result
	for _, inp := range inputs {
		if inp.value != "" {
			result[inp.name] = inp.value
		} else if inp.defVal != "" {
			result[inp.name] = inp.defVal
		}
	}

	return result, nil
}

// Confirm asks the user for confirmation.
func Confirm(message string, defaultVal bool) (bool, error) {
	if !isInteractive() {
		return defaultVal, nil
	}

	var result bool
	err := huh.NewConfirm().
		Title(message).
		Value(&result).
		Affirmative("Yes").
		Negative("No").
		Run()

	return result, err
}

// Select presents a list of options for the user to choose from.
func Select(title string, options []string) (string, error) {
	if !isInteractive() || len(options) == 0 {
		if len(options) > 0 {
			return options[0], nil
		}
		return "", nil
	}

	var result string
	var opts []huh.Option[string]
	for _, opt := range options {
		opts = append(opts, huh.NewOption(opt, opt))
	}

	err := huh.NewSelect[string]().
		Title(title).
		Options(opts...).
		Value(&result).
		Run()

	return result, err
}

// isInteractive returns true if stdin is a terminal.
func isInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
