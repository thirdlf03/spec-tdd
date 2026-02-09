package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitCommand(t *testing.T) {
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

	if err := initCmd.RunE(initCmd, []string{}); err != nil {
		t.Fatalf("initCmd error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, ".tdd", "config.yml")); err != nil {
		t.Fatalf("expected config file, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, ".tdd", "specs")); err != nil {
		t.Fatalf("expected specs dir, got %v", err)
	}
}
