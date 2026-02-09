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

var contextCommentPattern = regexp.MustCompile(`<!--\s*context:\s*(.+?)\s*-->`)

// SegmentMeta represents parsed segment metadata used internally.
type SegmentMeta struct {
	SegmentID   string   `json:"segment_id"`
	HeadingPath []string `json:"heading_path"`
	FilePath    string   `json:"file_path"`
}

// kireRawEntry matches kire's actual JSONL output format.
type kireRawEntry struct {
	Content  string          `json:"content"`
	Metadata kireRawMetadata `json:"metadata"`
}

type kireRawMetadata struct {
	Source       string   `json:"source"`
	SegmentIndex int      `json:"segment_index"`
	Filename     string   `json:"filename"`
	HeadingPath  []string `json:"heading_path"`
	TokenCount   int      `json:"token_count"`
	BlockCount   int      `json:"block_count"`
}

// ParseJSONL parses a kire JSONL metadata file and returns entries sorted by segment_index ascending.
func ParseJSONL(path string) ([]SegmentMeta, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, apperrors.Wrap("kire.ParseJSONL", fmt.Errorf("%s: %w", path, err))
	}
	defer f.Close()

	type indexedMeta struct {
		index int
		meta  SegmentMeta
	}

	var entries []indexedMeta
	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var raw kireRawEntry
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			return nil, apperrors.Wrap("kire.ParseJSONL", fmt.Errorf("line %d: %w", lineNum, err))
		}

		entries = append(entries, indexedMeta{
			index: raw.Metadata.SegmentIndex,
			meta: SegmentMeta{
				SegmentID:   fmt.Sprintf("seg-%04d", raw.Metadata.SegmentIndex),
				HeadingPath: raw.Metadata.HeadingPath,
				FilePath:    raw.Metadata.Filename,
			},
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, apperrors.Wrap("kire.ParseJSONL", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].index < entries[j].index
	})

	metas := make([]SegmentMeta, len(entries))
	for i, e := range entries {
		metas[i] = e.meta
	}

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
