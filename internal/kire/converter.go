package kire

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/thirdlf03/spec-tdd/internal/spec"
)

var (
	reqIDExtractPattern = regexp.MustCompile(`REQ-(\d{3})`)
	gwtGivenPattern     = regexp.MustCompile(`(?i)^[-*]?\s*given:\s*(.+)`)
	gwtWhenPattern      = regexp.MustCompile(`(?i)^[-*]?\s*when:\s*(.+)`)
	gwtThenPattern      = regexp.MustCompile(`(?i)^[-*]?\s*then:\s*(.+)`)
	questionsSectionRe  = regexp.MustCompile(`(?i)^#{2,3}\s+questions`)
	headingRe           = regexp.MustCompile(`^#+\s+`)
)

// ExtractReqID extracts the first REQ-### pattern from content.
// Returns empty string if not found.
func ExtractReqID(content string) string {
	matches := reqIDExtractPattern.FindString(content)
	return matches
}

// ExtractExamples extracts Given/When/Then example sets from content.
func ExtractExamples(content string) []spec.Example {
	lines := strings.Split(content, "\n")
	var examples []spec.Example

	for i := 0; i < len(lines); i++ {
		givenMatch := gwtGivenPattern.FindStringSubmatch(strings.TrimSpace(lines[i]))
		if givenMatch == nil {
			continue
		}

		// Look for When on the next line
		if i+1 >= len(lines) {
			continue
		}
		whenMatch := gwtWhenPattern.FindStringSubmatch(strings.TrimSpace(lines[i+1]))
		if whenMatch == nil {
			continue
		}

		// Look for Then on the line after
		if i+2 >= len(lines) {
			continue
		}
		thenMatch := gwtThenPattern.FindStringSubmatch(strings.TrimSpace(lines[i+2]))
		if thenMatch == nil {
			continue
		}

		examples = append(examples, spec.Example{
			Given: strings.TrimSpace(givenMatch[1]),
			When:  strings.TrimSpace(whenMatch[1]),
			Then:  strings.TrimSpace(thenMatch[1]),
		})

		i += 2 // skip the when/then lines
	}

	return examples
}

// ExtractQuestions extracts questions from content.
// Matches lines ending with '?' (excluding headings) and lines in a Questions section.
func ExtractQuestions(content string) []string {
	lines := strings.Split(content, "\n")
	var questions []string
	inQuestionsSection := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Check for Questions section header
		if questionsSectionRe.MatchString(trimmed) {
			inQuestionsSection = true
			continue
		}

		// If we hit another heading, leave the Questions section
		if inQuestionsSection && headingRe.MatchString(trimmed) {
			inQuestionsSection = false
			continue
		}

		if inQuestionsSection {
			// Strip leading bullet markers
			q := strings.TrimLeft(trimmed, "-* ")
			q = strings.TrimSpace(q)
			if q != "" {
				questions = append(questions, q)
			}
			continue
		}

		// Lines ending with '?' but not headings
		if strings.HasSuffix(trimmed, "?") || strings.HasSuffix(trimmed, "？") {
			if headingRe.MatchString(trimmed) {
				continue
			}
			questions = append(questions, trimmed)
		}
	}

	return questions
}

// ConvertToSpec converts a Segment into a spec.Spec.
// specDir is used to determine the next available REQ ID when auto-assigning.
func ConvertToSpec(seg *Segment, specDir string) (*spec.Spec, error) {
	if seg == nil {
		return nil, fmt.Errorf("segment is nil")
	}

	// Determine title from heading_path last element
	title := ""
	if len(seg.Meta.HeadingPath) > 0 {
		title = seg.Meta.HeadingPath[len(seg.Meta.HeadingPath)-1]
	}

	// Determine REQ ID
	id := ExtractReqID(seg.Content)
	if id == "" {
		nextID, err := spec.NextReqID(specDir)
		if err != nil {
			return nil, fmt.Errorf("auto-assign ID: %w", err)
		}
		id = nextID
	}

	// Extract examples and questions
	examples := ExtractExamples(seg.Content)
	// Assign example IDs
	for i := range examples {
		examples[i].ID = fmt.Sprintf("E%d", i+1)
	}

	questions := ExtractQuestions(seg.Content)

	// Build source info
	headingPath := make([]string, len(seg.Meta.HeadingPath))
	copy(headingPath, seg.Meta.HeadingPath)

	s := &spec.Spec{
		ID:        id,
		Title:     title,
		Examples:  examples,
		Questions: questions,
		Source: spec.SourceInfo{
			SegmentID:   seg.Meta.SegmentID,
			HeadingPath: headingPath,
		},
	}

	return s, nil
}
