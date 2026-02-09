package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReqAddCommand(t *testing.T) {
	tmpDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd error: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir error: %v", err)
	}

	reqAddTitle = "My requirement"
	reqAddID = ""

	if err := reqAddCmd.RunE(reqAddCmd, []string{}); err != nil {
		t.Fatalf("reqAddCmd error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, ".tdd", "specs", "REQ-001.yml")); err != nil {
		t.Fatalf("expected spec file, got %v", err)
	}
}
