package config

import (
	"errors"
	"os"
	"strings"

	"github.com/thirdlf03/spec-tdd/internal/apperrors"
	"go.yaml.in/yaml/v3"
)

const DefaultSpecConfigPath = ".tdd/config.yml"

// SpecConfig represents spec-driven TDD configuration stored in .tdd/config.yml
// Fields are flat to match the YAML DSL structure.
type SpecConfig struct {
	SpecDir         string `yaml:"specDir"`
	TestDir         string `yaml:"testDir"`
	Runner          string `yaml:"runner"`
	FileNamePattern string `yaml:"fileNamePattern"`
}

// DefaultSpecConfig returns the default spec configuration.
func DefaultSpecConfig() SpecConfig {
	return SpecConfig{
		SpecDir:         ".tdd/specs",
		TestDir:         "tests",
		Runner:          "vitest",
		FileNamePattern: "req-{{id}}-{{slug}}.test.ts",
	}
}

// LoadSpecConfig loads spec config from the given path.
// If the file does not exist, it returns defaults with loaded=false.
func LoadSpecConfig(path string) (SpecConfig, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultSpecConfig(), false, nil
		}
		return SpecConfig{}, false, apperrors.Wrap("config.LoadSpecConfig", err)
	}

	cfg := DefaultSpecConfig()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return SpecConfig{}, true, apperrors.Wrap("config.LoadSpecConfig", err)
	}

	if err := cfg.Validate(); err != nil {
		return SpecConfig{}, true, err
	}

	return cfg, true, nil
}

// SaveSpecConfig writes spec config to the given path.
func SaveSpecConfig(path string, cfg SpecConfig) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return apperrors.Wrap("config.SaveSpecConfig", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return apperrors.Wrap("config.SaveSpecConfig", err)
	}

	return nil
}

// Validate checks spec config values.
func (c SpecConfig) Validate() error {
	v := NewValidator()

	if strings.TrimSpace(c.SpecDir) == "" {
		v.AddError("specDir", "is required")
	}
	if strings.TrimSpace(c.TestDir) == "" {
		v.AddError("testDir", "is required")
	}
	if strings.TrimSpace(c.Runner) == "" {
		v.AddError("runner", "is required")
	} else if c.Runner != "vitest" && c.Runner != "jest" {
		v.AddError("runner", "must be vitest or jest")
	}
	if strings.TrimSpace(c.FileNamePattern) == "" {
		v.AddError("fileNamePattern", "is required")
	} else if !strings.Contains(c.FileNamePattern, "{{id}}") {
		v.AddError("fileNamePattern", "must include {{id}}")
	}

	if v.HasErrors() {
		return apperrors.Wrap("config.ValidateSpecConfig", v.Error())
	}

	return nil
}
