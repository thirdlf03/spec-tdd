package kire

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/thirdlf03/spec-tdd/internal/apperrors"
)

var contextCommentPattern = regexp.MustCompile(`<!--\s*kire:\s*(.+?)\s*-->`)

// SegmentMeta represents a single line from kire JSONL metadata.
type SegmentMeta struct {
	SegmentID   string   `json:"segment_id"`
	HeadingPath []string `json:"heading_path"`
	FilePath    string   `json:"file_path"`
}

// ParseJSONL parses a JSONL metadata file and returns entries sorted by segment_id ascending.
func ParseJSONL(path string) ([]SegmentMeta, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, apperrors.Wrap("kire.ParseJSONL", fmt.Errorf("%s: %w", path, err))
	}
	defer f.Close()

	var metas []SegmentMeta
	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var meta SegmentMeta
		if err := json.Unmarshal([]byte(line), &meta); err != nil {
			return nil, apperrors.Wrap("kire.ParseJSONL", fmt.Errorf("line %d: %w", lineNum, err))
		}
		metas = append(metas, meta)
	}

	if err := scanner.Err(); err != nil {
		return nil, apperrors.Wrap("kire.ParseJSONL", err)
	}

	sort.Slice(metas, func(i, j int) bool {
		return metas[i].SegmentID < metas[j].SegmentID
	})

	return metas, nil
}

// Segment represents a loaded segment with its content and context.
type Segment struct {
	Meta    SegmentMeta
	Content string
	Context string
}

// ReadSegment reads a segment markdown file from dir.
// Returns nil if the file does not exist (caller handles as warning).
func ReadSegment(dir string, meta SegmentMeta) (*Segment, error) {
	path := filepath.Join(dir, meta.FilePath)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, apperrors.Wrap("kire.ReadSegment", err)
	}

	content := string(data)
	context := extractContext(content)

	return &Segment{
		Meta:    meta,
		Content: content,
		Context: context,
	}, nil
}

func extractContext(content string) string {
	matches := contextCommentPattern.FindStringSubmatch(content)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}
