package trace

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/thirdlf03/spec-tdd/internal/apperrors"
	"github.com/thirdlf03/spec-tdd/internal/spec"
)

var testReqPattern = regexp.MustCompile(`(?m)^\s*(it|test)\s*\(\s*["'](REQ-\d+)`)

// Item represents a traceability entry.
type Item struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Expected int    `json:"expected"`
	Actual   int    `json:"actual"`
	Status   string `json:"status"`
}

// Report represents traceability results.
type Report struct {
	GeneratedAt time.Time `json:"generatedAt"`
	Items       []Item    `json:"items"`
}

// CountTestsByReq scans tests and counts REQ references in it()/test() names.
func CountTestsByReq(testDir string) (map[string]int, error) {
	counts := make(map[string]int)

	err := filepath.WalkDir(testDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !isTestFile(path) {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		matches := testReqPattern.FindAllSubmatch(data, -1)
		for _, m := range matches {
			if len(m) < 3 {
				continue
			}
			id := string(m[2])
			counts[id]++
		}

		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return counts, nil
		}
		return nil, apperrors.Wrap("trace.CountTestsByReq", err)
	}

	return counts, nil
}

// BuildReport builds a trace report from specs and test counts.
func BuildReport(specs []*spec.Spec, counts map[string]int) Report {
	items := make([]Item, 0, len(specs))
	for _, s := range specs {
		expected := len(s.Examples)
		actual := counts[s.ID]
		status := "OK"
		if expected == 0 {
			status = "MISSING"
		} else if actual == 0 {
			status = "MISSING"
		} else if actual < expected {
			status = "PARTIAL"
		}

		items = append(items, Item{
			ID:       s.ID,
			Title:    s.Title,
			Expected: expected,
			Actual:   actual,
			Status:   status,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})

	return Report{
		GeneratedAt: time.Now().UTC(),
		Items:       items,
	}
}

// ToJSON encodes the report to JSON.
func (r Report) ToJSON() ([]byte, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return nil, apperrors.Wrap("trace.ToJSON", err)
	}
	return data, nil
}

// ToMarkdown renders the report in Markdown.
func (r Report) ToMarkdown() string {
	var sb strings.Builder
	sb.WriteString("# Traceability Report\n\n")
	sb.WriteString(fmt.Sprintf("Generated: %s\n\n", r.GeneratedAt.Format(time.RFC3339)))
	sb.WriteString("| REQ ID | Title | Expected | Actual | Status |\n")
	sb.WriteString("| --- | --- | --- | --- | --- |\n")
	for _, item := range r.Items {
		sb.WriteString(fmt.Sprintf("| %s | %s | %d | %d | %s |\n", item.ID, escapePipes(item.Title), item.Expected, item.Actual, item.Status))
	}
	return sb.String()
}

func escapePipes(s string) string {
	return strings.ReplaceAll(s, "|", "\\|")
}

func isTestFile(path string) bool {
	name := filepath.Base(path)
	return strings.HasSuffix(name, ".test.ts") ||
		strings.HasSuffix(name, ".test.tsx") ||
		strings.HasSuffix(name, ".spec.ts") ||
		strings.HasSuffix(name, ".spec.tsx") ||
		strings.HasSuffix(name, ".test.js") ||
		strings.HasSuffix(name, ".spec.js") ||
		strings.HasSuffix(name, ".test.jsx") ||
		strings.HasSuffix(name, ".spec.jsx")
}
