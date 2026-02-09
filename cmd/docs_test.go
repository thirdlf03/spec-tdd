package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDocsCommand(t *testing.T) {
	// Test that docs command is properly initialized
	if docsCmd == nil {
		t.Fatal("docsCmd is nil")
	}

	if docsCmd.Use != "docs" {
		t.Errorf("docsCmd.Use = %q, want %q", docsCmd.Use, "docs")
	}
}

func TestDocsFlags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		defaultValue string
	}{
		{
			name:         "output flag",
			flagName:     "output",
			defaultValue: "./docs",
		},
		{
			name:         "format flag",
			flagName:     "format",
			defaultValue: "markdown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := docsCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Fatalf("flag %q not found", tt.flagName)
			}

			if flag.DefValue != tt.defaultValue {
				t.Errorf("flag %q default value = %q, want %q", tt.flagName, flag.DefValue, tt.defaultValue)
			}
		})
	}
}

func TestDocsGeneration(t *testing.T) {
	// Create a temporary directory for test output
	tmpDir := t.TempDir()

	tests := []struct {
		name       string
		format     string
		shouldFail bool
	}{
		{
			name:       "markdown format",
			format:     "markdown",
			shouldFail: false,
		},
		{
			name:       "md format",
			format:     "md",
			shouldFail: false,
		},
		{
			name:       "yaml format",
			format:     "yaml",
			shouldFail: false,
		},
		{
			name:       "invalid format",
			format:     "invalid",
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a subdirectory for this test
			outputDir := filepath.Join(tmpDir, tt.name)

			// Set the flags
			docsOutputDir = outputDir
			docsFormat = tt.format

			// Run the command
			err := docsCmd.RunE(docsCmd, []string{})

			if tt.shouldFail {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify output directory was created
			if _, err := os.Stat(outputDir); os.IsNotExist(err) {
				t.Errorf("output directory %q was not created", outputDir)
			}

			// Verify at least one file was generated
			entries, err := os.ReadDir(outputDir)
			if err != nil {
				t.Errorf("failed to read output directory: %v", err)
				return
			}

			if len(entries) == 0 {
				t.Error("no documentation files were generated")
			}
		})
	}
}
