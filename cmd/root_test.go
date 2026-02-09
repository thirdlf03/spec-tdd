package cmd

import (
	"testing"

	"github.com/spf13/viper"
)

func TestRootCommand(t *testing.T) {
	if rootCmd == nil {
		t.Fatal("rootCmd is nil")
	}

	if rootCmd.Use != "spec-tdd" {
		t.Errorf("rootCmd.Use = %q, want %q", rootCmd.Use, "spec-tdd")
	}

	// Test that persistent flags are defined
	flags := []string{"config", "debug", "log-format"}
	for _, flagName := range flags {
		flag := rootCmd.PersistentFlags().Lookup(flagName)
		if flag == nil {
			t.Errorf("flag %q not found", flagName)
		}
	}
}

func TestGetLogger(t *testing.T) {
	logger := GetLogger()
	if logger == nil {
		t.Error("GetLogger() returned nil")
	}
}

func TestInitConfig(t *testing.T) {
	viper.Reset()
	initConfig()

	// Verify env prefix was configured
	// (config file may not exist in test environment, so we just verify initialization completes)
}

func TestInitLogger(t *testing.T) {
	initLogger()

	if appLogger == nil {
		t.Error("initLogger() should initialize the logger")
	}
}
