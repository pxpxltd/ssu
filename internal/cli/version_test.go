package cli

import (
	"bytes"
	"runtime"
	"strings"
	"testing"
)

func TestVersionCmd(t *testing.T) {
	root := NewRootCmd("1.2.3", "abc1234", "2026-02-09")
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"version"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()

	for _, want := range []string{"1.2.3", "abc1234", "2026-02-09", runtime.Version()} {
		if !strings.Contains(out, want) {
			t.Errorf("version output missing %q\ngot: %s", want, out)
		}
	}
}

func TestVersionCmdNoArgs(t *testing.T) {
	root := NewRootCmd("1.0.0", "abc", "today")
	root.SetArgs([]string{"version", "extraarg"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for extra argument, got nil")
	}
}
