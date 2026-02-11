package spec

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/thirdlf03/spec-tdd/internal/apperrors"
	"go.yaml.in/yaml/v3"
)

var (
	reqIDPattern     = regexp.MustCompile(`^REQ-(\d+)$`)
	exampleIDPattern = regexp.MustCompile(`^E(\d+)$`)
)

// SourceInfo represents the origin of a spec imported from external tools.
type SourceInfo struct {
	SegmentID   string   `yaml:"segment_id,omitempty"`
	HeadingPath []string `yaml:"heading_path,omitempty"`
	FilePath    string   `yaml:"file_path,omitempty"`
}

// Spec represents a requirement spec file.
type Spec struct {
	ID          string     `yaml:"id"`
	Title       string     `yaml:"title"`
	Description string     `yaml:"description,omitempty"`
	Source      SourceInfo `yaml:"source,omitempty"`
	Depends     []string   `yaml:"depends,omitempty"`
	Examples    []Example  `yaml:"examples,omitempty"`
	Questions   []string   `yaml:"questions,omitempty"`
	Tags        []string   `yaml:"tags,omitempty"`
}

// Example represents a Given/When/Then example.
type Example struct {
	ID    string `yaml:"id,omitempty"`
	Given string `yaml:"given"`
	When  string `yaml:"when"`
	Then  string `yaml:"then"`
}

// Validate checks required fields.
func (s *Spec) Validate() error {
	if strings.TrimSpace(s.ID) == "" {
		return apperrors.New("spec.Validate", apperrors.ErrInvalidInput, "id is required")
	}
	if !reqIDPattern.MatchString(s.ID) {
		return apperrors.New("spec.Validate", apperrors.ErrInvalidInput, "id must match REQ-###")
	}
	if strings.TrimSpace(s.Title) == "" {
		return apperrors.New("spec.Validate", apperrors.ErrInvalidInput, "title is required")
	}
	for i, ex := range s.Examples {
		if strings.TrimSpace(ex.Given) == "" || strings.TrimSpace(ex.When) == "" || strings.TrimSpace(ex.Then) == "" {
			return apperrors.New("spec.Validate", apperrors.ErrInvalidInput, fmt.Sprintf("example %d must include given/when/then", i+1))
		}
	}

	// Validate Depends
	seen := make(map[string]bool, len(s.Depends))
	for _, dep := range s.Depends {
		if !reqIDPattern.MatchString(dep) {
			return apperrors.New("spec.Validate", apperrors.ErrInvalidInput,
				fmt.Sprintf("depends entry %q must match REQ-### format", dep))
		}
		if dep == s.ID {
			return apperrors.New("spec.Validate", apperrors.ErrInvalidInput,
				fmt.Sprintf("depends entry %q is a self-reference", dep))
		}
		if seen[dep] {
			return apperrors.New("spec.Validate", apperrors.ErrInvalidInput,
				fmt.Sprintf("duplicate depends entry %q", dep))
		}
		seen[dep] = true
	}

	return nil
}

// ValidateDependsRefs checks that all referenced dependencies exist.
// Unlike ValidateDependsGraph, it does not check for cycles.
func ValidateDependsRefs(specs []*Spec) error {
	ids := make(map[string]bool, len(specs))
	for _, s := range specs {
		ids[s.ID] = true
	}

	for _, s := range specs {
		for _, dep := range s.Depends {
			if !ids[dep] {
				return apperrors.New("spec.ValidateDependsRefs", apperrors.ErrInvalidInput,
					fmt.Sprintf("%s depends on %s which does not exist", s.ID, dep))
			}
		}
	}
	return nil
}

// ValidateDependsGraph checks the dependency graph across all specs for
// missing references and cycles. Uses DFS 3-color (white/gray/black).
func ValidateDependsGraph(specs []*Spec) error {
	if err := ValidateDependsRefs(specs); err != nil {
		return err
	}

	// Build adjacency list
	adj := make(map[string][]string, len(specs))
	for _, s := range specs {
		adj[s.ID] = s.Depends
	}

	// DFS cycle detection: 0=white, 1=gray, 2=black
	color := make(map[string]int, len(specs))
	var cyclePath []string

	var dfs func(id string) bool
	dfs = func(id string) bool {
		color[id] = 1 // gray
		cyclePath = append(cyclePath, id)
		for _, dep := range adj[id] {
			switch color[dep] {
			case 1: // gray = cycle
				cyclePath = append(cyclePath, dep)
				return true
			case 0: // white = unvisited
				if dfs(dep) {
					return true
				}
			}
		}
		cyclePath = cyclePath[:len(cyclePath)-1]
		color[id] = 2 // black
		return false
	}

	for _, s := range specs {
		if color[s.ID] == 0 {
			cyclePath = nil
			if dfs(s.ID) {
				return apperrors.New("spec.ValidateDependsGraph", apperrors.ErrInvalidInput,
					fmt.Sprintf("dependency cycle detected: %s", strings.Join(cyclePath, " -> ")))
			}
		}
	}

	return nil
}

// Normalize fills missing example IDs in-memory.
func (s *Spec) Normalize() {
	next := NextExampleID(s)
	for i := range s.Examples {
		if strings.TrimSpace(s.Examples[i].ID) == "" {
			s.Examples[i].ID = next
			next = NextExampleID(s)
		}
	}
}

// Load reads a spec from disk.
func Load(path string) (*Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, apperrors.Wrap("spec.Load", err)
	}

	var s Spec
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, apperrors.Wrap("spec.Load", err)
	}

	if strings.TrimSpace(s.ID) == "" {
		s.ID = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}

	if err := s.Validate(); err != nil {
		return nil, err
	}

	return &s, nil
}

// Save writes a spec to disk.
func Save(path string, s *Spec) error {
	if s == nil {
		return apperrors.New("spec.Save", apperrors.ErrInvalidInput, "spec is nil")
	}
	if err := s.Validate(); err != nil {
		return err
	}

	data, err := yaml.Marshal(s)
	if err != nil {
		return apperrors.Wrap("spec.Save", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return apperrors.Wrap("spec.Save", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return apperrors.Wrap("spec.Save", err)
	}

	return nil
}

// ListFiles returns spec files in the spec directory.
func ListFiles(specDir string) ([]string, error) {
	entries, err := os.ReadDir(specDir)
	if err != nil {
		return nil, apperrors.Wrap("spec.ListFiles", err)
	}

	files := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".yml" && filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		files = append(files, filepath.Join(specDir, entry.Name()))
	}

	sort.Strings(files)
	return files, nil
}

// LoadAll loads all specs from a directory.
func LoadAll(specDir string) ([]*Spec, error) {
	files, err := ListFiles(specDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []*Spec{}, nil
		}
		return nil, err
	}

	out := make([]*Spec, 0, len(files))
	for _, path := range files {
		s, err := Load(path)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}

	sort.Slice(out, func(i, j int) bool {
		return reqIDLess(out[i].ID, out[j].ID)
	})

	return out, nil
}

// NextReqID returns the next available REQ-### ID in the directory.
func NextReqID(specDir string) (string, error) {
	files, err := ListFiles(specDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "REQ-001", nil
		}
		return "", err
	}

	max := 0
	for _, path := range files {
		base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		if matches := reqIDPattern.FindStringSubmatch(base); len(matches) == 2 {
			val, _ := strconv.Atoi(matches[1])
			if val > max {
				max = val
			}
			continue
		}
		s, err := Load(path)
		if err != nil {
			return "", err
		}
		if matches := reqIDPattern.FindStringSubmatch(s.ID); len(matches) == 2 {
			val, _ := strconv.Atoi(matches[1])
			if val > max {
				max = val
			}
		}
	}

	return fmt.Sprintf("REQ-%03d", max+1), nil
}

// NextExampleID returns the next example ID for the spec.
func NextExampleID(s *Spec) string {
	max := 0
	for _, ex := range s.Examples {
		if matches := exampleIDPattern.FindStringSubmatch(strings.TrimSpace(ex.ID)); len(matches) == 2 {
			val, _ := strconv.Atoi(matches[1])
			if val > max {
				max = val
			}
		}
	}
	return fmt.Sprintf("E%d", max+1)
}

func reqIDLess(a, b string) bool {
	am := reqIDPattern.FindStringSubmatch(a)
	bm := reqIDPattern.FindStringSubmatch(b)
	if len(am) == 2 && len(bm) == 2 {
		ai, _ := strconv.Atoi(am[1])
		bi, _ := strconv.Atoi(bm[1])
		return ai < bi
	}
	return a < b
}
