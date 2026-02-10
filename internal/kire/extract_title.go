package kire

import (
	"fmt"
	"regexp"
	"strings"
)

var reqIDWithTitlePattern = regexp.MustCompile(`###\s+REQ-(\d{3}):\s*(.+)`)

// ExtractReqIDWithTitle は content 内の `### REQ-XXX: タイトル` パターンから
// REQ-ID とタイトルを同時抽出する。パターンが見つからない場合は空文字列を返す。
func ExtractReqIDWithTitle(content string) (string, string) {
	matches := reqIDWithTitlePattern.FindStringSubmatch(content)
	if len(matches) < 3 {
		return "", ""
	}
	id := fmt.Sprintf("REQ-%s", matches[1])
	title := strings.TrimSpace(matches[2])
	return id, title
}

// CheckDuplicateReqIDs は ID リスト内の重複を検出してエラーを返す。
func CheckDuplicateReqIDs(ids []string) error {
	seen := make(map[string]int)
	var duplicates []string

	for i, id := range ids {
		if prev, ok := seen[id]; ok {
			duplicates = append(duplicates, fmt.Sprintf("%s (index %d and %d)", id, prev, i))
		}
		seen[id] = i
	}

	if len(duplicates) > 0 {
		return fmt.Errorf("duplicate REQ-IDs found: %s", strings.Join(duplicates, ", "))
	}
	return nil
}
