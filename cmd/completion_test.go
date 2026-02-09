package cmd

import (
	"bytes"
	"testing"
)

func TestCompletionCommand(t *testing.T) {
	if completionCmd == nil {
		t.Fatal("completionCmd is nil")
	}

	if completionCmd.Use != "completion [bash|zsh|fish|powershell]" {
		t.Errorf("completionCmd.Use = %q, want %q", completionCmd.Use, "completion [bash|zsh|fish|powershell]")
	}
}

func TestCompletionValidArgs(t *testing.T) {
	validArgs := []string{"bash", "zsh", "fish", "powershell"}

	if len(completionCmd.ValidArgs) != len(validArgs) {
		t.Errorf("ValidArgs length = %d, want %d", len(completionCmd.ValidArgs), len(validArgs))
	}

	for i, arg := range validArgs {
		if completionCmd.ValidArgs[i] != arg {
			t.Errorf("ValidArgs[%d] = %q, want %q", i, completionCmd.ValidArgs[i], arg)
		}
	}
}

func TestCompletionOutput(t *testing.T) {
	shells := []string{"bash", "zsh", "fish", "powershell"}

	for _, shell := range shells {
		t.Run(shell, func(t *testing.T) {
			var buf bytes.Buffer
			completionCmd.SetOut(&buf)
			defer completionCmd.SetOut(nil)

			err := completionCmd.RunE(completionCmd, []string{shell})
			if err != nil {
				t.Fatalf("completion %s returned error: %v", shell, err)
			}

			if buf.Len() == 0 {
				t.Errorf("completion %s produced empty output", shell)
			}
		})
	}
}
