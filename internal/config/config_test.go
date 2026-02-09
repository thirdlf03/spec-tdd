package config

import (
	"testing"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				App: AppConfig{
					Name:  "test-app",
					Debug: false,
				},
			},
			wantErr: false,
		},
		{
			name: "empty name",
			config: &Config{
				App: AppConfig{
					Name:  "",
					Debug: false,
				},
			},
			wantErr: true,
		},
		{
			name: "debug enabled",
			config: &Config{
				App: AppConfig{
					Name:  "test-app",
					Debug: true,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	if cfg.App.Name == "" {
		t.Error("DefaultConfig() App.Name is empty")
	}

	if cfg.App.Name != "spec-tdd" {
		t.Errorf("DefaultConfig() App.Name = %q, want %q", cfg.App.Name, "spec-tdd")
	}

	if cfg.App.Debug {
		t.Error("DefaultConfig() App.Debug should be false by default")
	}

	// Verify default config is valid
	if err := cfg.Validate(); err != nil {
		t.Errorf("DefaultConfig() should be valid, got error: %v", err)
	}
}

func TestAppConfig(t *testing.T) {
	tests := []struct {
		name   string
		config AppConfig
		valid  bool
	}{
		{
			name: "minimal valid config",
			config: AppConfig{
				Name:  "spec-tdd",
				Debug: false,
			},
			valid: true,
		},
		{
			name: "debug mode enabled",
			config: AppConfig{
				Name:  "spec-tdd",
				Debug: true,
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{App: tt.config}
			err := cfg.Validate()
			if tt.valid && err != nil {
				t.Errorf("expected valid config, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("expected invalid config, got nil error")
			}
		})
	}
}
