package kire

import (
	"testing"
)

func TestExtractReqIDWithTitle(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantID    string
		wantTitle string
	}{
		{
			name:      "extracts REQ-ID and title from heading pattern",
			content:   "# 概要\n\n### REQ-001: ユーザーログイン\n\n説明文。",
			wantID:    "REQ-001",
			wantTitle: "ユーザーログイン",
		},
		{
			name:      "extracts with extra spaces around title",
			content:   "### REQ-042:  ロックアウト機能  \n",
			wantID:    "REQ-042",
			wantTitle: "ロックアウト機能",
		},
		{
			name:      "no pattern returns empty",
			content:   "# Login\n\nただの説明文。\n",
			wantID:    "",
			wantTitle: "",
		},
		{
			name:      "REQ-ID without colon title returns ID only",
			content:   "REQ-005\n\n説明文。",
			wantID:    "",
			wantTitle: "",
		},
		{
			name:      "multiple patterns returns first",
			content:   "### REQ-001: ログイン\n\n### REQ-002: ログアウト\n",
			wantID:    "REQ-001",
			wantTitle: "ログイン",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotTitle := ExtractReqIDWithTitle(tt.content)
			if gotID != tt.wantID {
				t.Errorf("ExtractReqIDWithTitle() ID = %q, want %q", gotID, tt.wantID)
			}
			if gotTitle != tt.wantTitle {
				t.Errorf("ExtractReqIDWithTitle() Title = %q, want %q", gotTitle, tt.wantTitle)
			}
		})
	}
}

func TestCheckDuplicateReqIDs(t *testing.T) {
	tests := []struct {
		name    string
		ids     []string
		wantErr bool
		wantMsg string
	}{
		{
			name:    "no duplicates",
			ids:     []string{"REQ-001", "REQ-002", "REQ-003"},
			wantErr: false,
		},
		{
			name:    "with duplicates",
			ids:     []string{"REQ-001", "REQ-002", "REQ-001"},
			wantErr: true,
			wantMsg: "REQ-001",
		},
		{
			name:    "empty list",
			ids:     []string{},
			wantErr: false,
		},
		{
			name:    "single item",
			ids:     []string{"REQ-001"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckDuplicateReqIDs(tt.ids)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantMsg != "" {
					errMsg := err.Error()
					if !containsSubstring(errMsg, tt.wantMsg) {
						t.Errorf("error message %q does not contain %q", errMsg, tt.wantMsg)
					}
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsCheck(s, sub))
}

func containsCheck(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
