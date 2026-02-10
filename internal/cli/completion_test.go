package cli

import (
	"bytes"
	"testing"
)

func TestCompletionBash(t *testing.T) {
	root := NewRootCmd("dev", "test", "now")
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"completion", "bash"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	// Bash completion writes to os.Stdout directly, but we can verify no error.
	// The script is written to stdout so buf may be empty -- the key is no error.
	_ = out
}

func TestCompletionZsh(t *testing.T) {
	root := NewRootCmd("dev", "test", "now")
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"completion", "zsh"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompletionFish(t *testing.T) {
	root := NewRootCmd("dev", "test", "now")
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"completion", "fish"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompletionInvalidShell(t *testing.T) {
	root := NewRootCmd("dev", "test", "now")
	root.SetArgs([]string{"completion", "invalid"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for invalid shell, got nil")
	}
}

func TestCompletionNoArgs(t *testing.T) {
	root := NewRootCmd("dev", "test", "now")
	root.SetArgs([]string{"completion"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for missing shell argument, got nil")
	}
}
