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
	"fmt"
	"sort"
	"strings"

	"github.com/cloudygreybeard/hack/internal/log"
)

// ResolvedPattern holds a pattern and its resolution provenance.
type ResolvedPattern struct {
	Pattern    Pattern
	Source     string // "requested", "inherited (via <name>)", "inherited (via patternSelector)"
}

// Resolve loads a pattern and recursively resolves its inheritance chain.
// Returns patterns in application order (bases first, requested last).
// Detects and rejects cycles.
func Resolve(patternsDir, patternName string) ([]ResolvedPattern, error) {
	allPatterns, err := List(patternsDir)
	if err != nil {
		return nil, fmt.Errorf("listing patterns: %w", err)
	}

	patternsByName := make(map[string]Pattern, len(allPatterns))
	for _, p := range allPatterns {
		patternsByName[p.Name] = p
	}

	root, ok := patternsByName[patternName]
	if !ok {
		return nil, fmt.Errorf("pattern %q not found", patternName)
	}

	// Build the dependency graph via DFS
	graph := make(map[string][]string) // pattern -> patterns it depends on
	sources := make(map[string]string) // pattern -> provenance description
	sources[patternName] = "requested"

	if err := buildGraph(root, patternsByName, graph, sources, nil); err != nil {
		return nil, err
	}

	// Topological sort
	order, err := topoSort(graph, patternName)
	if err != nil {
		return nil, err
	}

	// Sort patterns at the same topological depth by weight, then alphabetically
	result := make([]ResolvedPattern, 0, len(order))
	for _, name := range order {
		p, ok := patternsByName[name]
		if !ok {
			return nil, fmt.Errorf("resolved pattern %q not found", name)
		}
		result = append(result, ResolvedPattern{
			Pattern: p,
			Source:  sources[name],
		})
	}

	// Stable sort by weight (topoSort already handles dependency ordering)
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Pattern.Weight < result[j].Pattern.Weight
	})

	for _, rp := range result {
		log.Verbose("  %s (weight: %d, %s)", rp.Pattern.Name, rp.Pattern.Weight, rp.Source)
	}

	return result, nil
}

// buildGraph recursively resolves inherits and builds the dependency graph.
// visiting tracks the current DFS path for cycle detection.
func buildGraph(p Pattern, all map[string]Pattern, graph map[string][]string, sources map[string]string, visiting []string) error {
	// Cycle detection: check if we're already visiting this pattern
	for _, v := range visiting {
		if v == p.Name {
			cycle := append(visiting, p.Name)
			return fmt.Errorf("cycle detected in pattern inheritance: %s", strings.Join(cycle, " -> "))
		}
	}

	visiting = append(visiting, p.Name)

	if _, exists := graph[p.Name]; exists {
		return nil // already resolved
	}

	var deps []string

	for _, inh := range p.Inherits {
		if inh.Pattern != "" {
			dep, ok := all[inh.Pattern]
			if !ok {
				return fmt.Errorf("pattern %q inherits %q, but it was not found", p.Name, inh.Pattern)
			}
			deps = append(deps, dep.Name)
			if _, ok := sources[dep.Name]; !ok {
				sources[dep.Name] = fmt.Sprintf("inherited (via pattern: %s)", inh.Pattern)
			}
			if err := buildGraph(dep, all, graph, sources, visiting); err != nil {
				return err
			}
		}

		if inh.PatternSelector != nil {
			matched := selectPatterns(all, inh.PatternSelector.MatchLabels)
			for _, dep := range matched {
				if dep.Name == p.Name {
					continue // don't inherit yourself
				}
				deps = append(deps, dep.Name)
				if _, ok := sources[dep.Name]; !ok {
					sources[dep.Name] = fmt.Sprintf("inherited (via patternSelector: %s)",
						formatMatchLabels(inh.PatternSelector.MatchLabels))
				}
				if err := buildGraph(dep, all, graph, sources, visiting); err != nil {
					return err
				}
			}
		}
	}

	graph[p.Name] = deps
	return nil
}

// selectPatterns returns all patterns whose labels match the selector.
func selectPatterns(all map[string]Pattern, matchLabels map[string]string) []Pattern {
	var result []Pattern
	for _, p := range all {
		if labelsMatch(p.Labels, matchLabels) {
			result = append(result, p)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// labelsMatch returns true if all matchLabels are present in labels.
func labelsMatch(labels, matchLabels map[string]string) bool {
	if len(matchLabels) == 0 {
		return false // empty selector matches nothing, not everything
	}
	if labels == nil {
		return false
	}
	for k, v := range matchLabels {
		if labels[k] != v {
			return false
		}
	}
	return true
}

// topoSort performs a topological sort starting from root.
// Returns an error if a cycle is detected.
func topoSort(graph map[string][]string, root string) ([]string, error) {
	visited := make(map[string]bool)
	inStack := make(map[string]bool)
	var order []string

	var visit func(node string, path []string) error
	visit = func(node string, path []string) error {
		if inStack[node] {
			cycle := append(path, node)
			return fmt.Errorf("cycle detected in pattern inheritance: %s", strings.Join(cycle, " -> "))
		}
		if visited[node] {
			return nil
		}

		inStack[node] = true
		path = append(path, node)

		for _, dep := range graph[node] {
			if err := visit(dep, path); err != nil {
				return err
			}
		}

		inStack[node] = false
		visited[node] = true
		order = append(order, node)
		return nil
	}

	if err := visit(root, nil); err != nil {
		return nil, err
	}

	return order, nil
}

func formatMatchLabels(labels map[string]string) string {
	parts := make([]string, 0, len(labels))
	for k, v := range labels {
		parts = append(parts, k+"="+v)
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}
