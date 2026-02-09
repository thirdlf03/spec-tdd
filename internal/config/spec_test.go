package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultSpecConfig(t *testing.T) {
	cfg := DefaultSpecConfig()
	if cfg.SpecDir == "" || cfg.TestDir == "" || cfg.Runner == "" || cfg.FileNamePattern == "" {
		t.Fatal("default spec config should not have empty fields")
	}
}

func TestSpecConfigValidate(t *testing.T) {
	valid := DefaultSpecConfig()
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid config, got error: %v", err)
	}

	invalid := valid
	invalid.Runner = "unknown"
	if err := invalid.Validate(); err == nil {
		t.Error("expected error for invalid runner")
	}
}

func TestLoadSpecConfigMissingFile(t *testing.T) {
	cfg, loaded, err := LoadSpecConfig(filepath.Join(t.TempDir(), "missing.yml"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if loaded {
		t.Error("expected loaded=false for missing file")
	}
	if cfg.SpecDir == "" {
		t.Error("expected default config")
	}
}

func TestSaveSpecConfig(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.yml")
	cfg := DefaultSpecConfig()

	if err := SaveSpecConfig(path, cfg); err != nil {
		t.Fatalf("SaveSpecConfig error: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected config file to exist, got %v", err)
	}

	loaded, _, err := LoadSpecConfig(path)
	if err != nil {
		t.Fatalf("LoadSpecConfig error: %v", err)
	}
	if loaded.SpecDir != cfg.SpecDir {
		t.Fatalf("expected SpecDir %q, got %q", cfg.SpecDir, loaded.SpecDir)
	}
}
