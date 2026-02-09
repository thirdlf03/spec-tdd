package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	if versionCmd == nil {
		t.Fatal("versionCmd is nil")
	}

	if versionCmd.Use != "version" {
		t.Errorf("versionCmd.Use = %q, want %q", versionCmd.Use, "version")
	}
}

func TestVersionVariables(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{
			name: "default version",
			got:  Version,
			want: "dev",
		},
		{
			name: "default commit",
			got:  Commit,
			want: "none",
		},
		{
			name: "default build date",
			got:  BuildDate,
			want: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestVersionOutput(t *testing.T) {
	var buf bytes.Buffer
	versionCmd.SetOut(&buf)
	defer versionCmd.SetOut(nil)

	versionCmd.Run(versionCmd, []string{})

	output := buf.String()
	expectedFields := []string{"Version:", "Commit:", "Build Date:", "Go Version:", "Platform:"}
	for _, field := range expectedFields {
		if !strings.Contains(output, field) {
			t.Errorf("version output missing %q, got: %s", field, output)
		}
	}
}
