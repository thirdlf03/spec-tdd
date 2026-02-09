package logger

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		cfg    *Config
		logMsg string
		want   string
	}{
		{
			name: "default text format",
			cfg: &Config{
				Level:  slog.LevelInfo,
				Format: "text",
				Writer: &bytes.Buffer{},
			},
			logMsg: "test message",
			want:   "test message",
		},
		{
			name: "json format",
			cfg: &Config{
				Level:  slog.LevelInfo,
				Format: "json",
				Writer: &bytes.Buffer{},
			},
			logMsg: "test message",
			want:   `"msg":"test message"`,
		},
		{
			name:   "nil config uses default",
			cfg:    nil,
			logMsg: "test message",
			want:   "test message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if tt.cfg != nil {
				tt.cfg.Writer = &buf
			}

			logger := New(tt.cfg)
			if tt.cfg == nil {
				// For nil config test, we can't capture output easily
				// Just verify it doesn't panic
				logger.Info(tt.logMsg)
				return
			}

			logger.Info(tt.logMsg)

			got := buf.String()
			if !strings.Contains(got, tt.want) {
				t.Errorf("log output = %q, want to contain %q", got, tt.want)
			}
		})
	}
}

func TestNewFromFlags(t *testing.T) {
	tests := []struct {
		name   string
		debug  bool
		format string
		want   slog.Level
	}{
		{
			name:   "debug enabled",
			debug:  true,
			format: "text",
			want:   slog.LevelDebug,
		},
		{
			name:   "debug disabled",
			debug:  false,
			format: "text",
			want:   slog.LevelInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewFromFlags(tt.debug, tt.format)
			if logger == nil {
				t.Error("logger is nil")
			}
		})
	}
}

func TestWithComponent(t *testing.T) {
	var buf bytes.Buffer
	cfg := &Config{
		Level:  slog.LevelInfo,
		Format: "json",
		Writer: &buf,
	}

	logger := New(cfg).WithComponent("test-component")
	logger.Info("test message")

	got := buf.String()
	if !strings.Contains(got, "test-component") {
		t.Errorf("log output = %q, want to contain component name", got)
	}
}

func TestWithFields(t *testing.T) {
	var buf bytes.Buffer
	cfg := &Config{
		Level:  slog.LevelInfo,
		Format: "json",
		Writer: &buf,
	}

	logger := New(cfg).WithFields(
		slog.String("key1", "value1"),
		slog.Int("key2", 42),
	)
	logger.Info("test message")

	got := buf.String()
	if strings.Contains(got, `"fields"`) {
		t.Errorf("WithFields should not nest attrs under 'fields' key, got: %s", got)
	}
	if !strings.Contains(got, `"key1":"value1"`) {
		t.Errorf("WithFields output missing key1, got: %s", got)
	}
	if !strings.Contains(got, `"key2":42`) {
		t.Errorf("WithFields output missing key2, got: %s", got)
	}
}
