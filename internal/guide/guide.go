package guide

import (
	"fmt"
	"strings"

	"github.com/thirdlf03/spec-tdd/internal/apperrors"
	"github.com/thirdlf03/spec-tdd/internal/scaffold"
	"github.com/thirdlf03/spec-tdd/internal/spec"
)

// GuideData holds data needed to render the implementation guide.
type GuideData struct {
	Specs         []*spec.Spec
	Order         []*spec.Spec
	Prerequisites map[string][]string // ID -> direct dependencies
	DependedBy    map[string][]string // ID -> reverse dependencies
}

// TopologicalSort returns specs in dependency order using Kahn's algorithm.
// Specs at the same level are sorted by REQ-ID numerically.
// Returns an error if a cycle is detected.
func TopologicalSort(specs []*spec.Spec) ([]*spec.Spec, error) {
	specMap := make(map[string]*spec.Spec, len(specs))
	inDegree := make(map[string]int, len(specs))
	adj := make(map[string][]string, len(specs))

	for _, s := range specs {
		specMap[s.ID] = s
		inDegree[s.ID] = 0
	}

	for _, s := range specs {
		for _, dep := range s.Depends {
			if _, ok := specMap[dep]; ok {
				adj[dep] = append(adj[dep], s.ID)
				inDegree[s.ID]++
			}
		}
	}

	// Find all nodes with in-degree 0
	var queue []string
	for _, s := range specs {
		if inDegree[s.ID] == 0 {
			queue = append(queue, s.ID)
		}
	}

	var result []*spec.Spec
	for len(queue) > 0 {
		// Sort queue for deterministic order (by REQ-ID number)
		sortQueue(queue)

		id := queue[0]
		queue = queue[1:]
		result = append(result, specMap[id])

		for _, next := range adj[id] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	if len(result) != len(specs) {
		return nil, apperrors.New("guide.TopologicalSort", apperrors.ErrInvalidInput,
			"dependency cycle detected, cannot determine implementation order")
	}

	return result, nil
}

// BuildDependedByMap builds a reverse dependency map.
func BuildDependedByMap(specs []*spec.Spec) map[string][]string {
	idSet := make(map[string]bool, len(specs))
	for _, s := range specs {
		idSet[s.ID] = true
	}

	result := make(map[string][]string)
	for _, s := range specs {
		for _, dep := range s.Depends {
			if idSet[dep] {
				result[dep] = append(result[dep], s.ID)
			}
		}
	}
	return result
}

// RenderGuide generates the GUIDE.md content.
func RenderGuide(data GuideData, testDir, fileNamePattern string) string {
	var sb strings.Builder

	// Overview
	sb.WriteString("# Implementation Guide\n\n")
	sb.WriteString(fmt.Sprintf("Total requirements: %d\n\n", len(data.Specs)))

	// Prerequisites summary
	sb.WriteString("## Prerequisites\n\n")
	hasPrereqs := false
	for _, s := range data.Order {
		if len(s.Depends) > 0 {
			hasPrereqs = true
			sb.WriteString(fmt.Sprintf("- **%s** (%s) requires: %s\n",
				s.ID, s.Title, strings.Join(s.Depends, ", ")))
		}
	}
	if !hasPrereqs {
		sb.WriteString("No dependencies detected.\n")
	}
	sb.WriteString("\n")

	// Dependency Graph (ASCII tree)
	sb.WriteString("## Dependency Graph\n\n")
	sb.WriteString("```\n")
	roots := findRoots(data.Order)
	visited := make(map[string]bool)
	for _, r := range roots {
		renderTree(&sb, r.ID, data.DependedBy, visited, "", true)
	}
	// Render any isolated nodes not yet visited
	for _, s := range data.Order {
		if !visited[s.ID] {
			sb.WriteString(s.ID + "\n")
		}
	}
	sb.WriteString("```\n\n")

	// Implementation Order
	sb.WriteString("## Implementation Order\n\n")
	for i, s := range data.Order {
		sb.WriteString(fmt.Sprintf("%d. **%s**: %s\n", i+1, s.ID, s.Title))
	}
	sb.WriteString("\n")

	// Feature Details
	sb.WriteString("## Feature Details\n\n")
	for _, s := range data.Order {
		sb.WriteString(fmt.Sprintf("### %s: %s\n\n", s.ID, s.Title))

		if len(s.Depends) > 0 {
			sb.WriteString(fmt.Sprintf("**Depends on:** %s\n\n", strings.Join(s.Depends, ", ")))
		}

		depBy := data.DependedBy[s.ID]
		if len(depBy) > 0 {
			sb.WriteString(fmt.Sprintf("**Required by:** %s\n\n", strings.Join(depBy, ", ")))
		}

		// Test file path
		slug := scaffold.Slugify(s.Title)
		testFile := scaffold.ApplyPattern(fileNamePattern, s.ID, slug)
		sb.WriteString(fmt.Sprintf("**Test file:** `%s/%s`\n\n", testDir, testFile))

		if len(s.Examples) > 0 {
			sb.WriteString("**Examples:**\n")
			for _, ex := range s.Examples {
				id := ex.ID
				if id == "" {
					id = "E?"
				}
				sb.WriteString(fmt.Sprintf("- %s: Given %s / When %s / Then %s\n", id, ex.Given, ex.When, ex.Then))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// findRoots returns specs that have no dependencies (root nodes).
func findRoots(specs []*spec.Spec) []*spec.Spec {
	var roots []*spec.Spec
	for _, s := range specs {
		if len(s.Depends) == 0 {
			roots = append(roots, s)
		}
	}
	return roots
}

// renderTree renders an ASCII tree for dependency visualization.
func renderTree(sb *strings.Builder, id string, dependedBy map[string][]string, visited map[string]bool, prefix string, isLast bool) {
	if visited[id] {
		sb.WriteString(prefix)
		if isLast {
			sb.WriteString("└── ")
		} else {
			sb.WriteString("├── ")
		}
		sb.WriteString(id + " (see above)\n")
		return
	}
	visited[id] = true

	if prefix == "" {
		sb.WriteString(id + "\n")
	} else {
		if isLast {
			sb.WriteString(prefix + "└── " + id + "\n")
		} else {
			sb.WriteString(prefix + "├── " + id + "\n")
		}
	}

	children := dependedBy[id]
	childPrefix := prefix
	if prefix != "" {
		if isLast {
			childPrefix += "    "
		} else {
			childPrefix += "│   "
		}
	}

	for i, child := range children {
		isChildLast := i == len(children)-1
		renderTree(sb, child, dependedBy, visited, childPrefix, isChildLast)
	}
}

// sortQueue sorts REQ-IDs in numeric order.
func sortQueue(ids []string) {
	for i := 1; i < len(ids); i++ {
		for j := i; j > 0 && reqIDNum(ids[j-1]) > reqIDNum(ids[j]); j-- {
			ids[j-1], ids[j] = ids[j], ids[j-1]
		}
	}
}

// reqIDNum extracts the numeric part of a REQ-### ID for sorting.
func reqIDNum(id string) int {
	if len(id) > 4 && id[:4] == "REQ-" {
		n := 0
		for _, c := range id[4:] {
			if c >= '0' && c <= '9' {
				n = n*10 + int(c-'0')
			}
		}
		return n
	}
	return 999999
}
