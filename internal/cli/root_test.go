package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCmdHelp(t *testing.T) {
	root := NewRootCmd("dev", "test", "now")
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	for _, want := range []string{"status", "update", "push", "rollback", "backup", "export", "import", "config", "version", "completion"} {
		if !strings.Contains(out, want) {
			t.Errorf("help output missing %q\ngot: %s", want, out)
		}
	}
}

func TestRootCmdImplementedSubcommands(t *testing.T) {
	// All commands with real implementations
	cmds := []string{"status", "update", "push", "backup", "rollback"}

	for _, sub := range cmds {
		t.Run(sub, func(t *testing.T) {
			root := NewRootCmd("dev", "test", "now")
			var buf bytes.Buffer
			root.SetOut(&buf)
			root.SetArgs([]string{sub})

			// Should run without error (may produce output depending on git state)
			if err := root.Execute(); err != nil {
				t.Fatalf("unexpected error running %q: %v", sub, err)
			}
		})
	}
}

func TestGlobalFlags(t *testing.T) {
	root := NewRootCmd("dev", "test", "now")

	flags := []struct {
		name      string
		shorthand string
	}{
		{"verbose", "v"},
		{"dry-run", "n"},
		{"auto", "a"},
		{"jobs", "j"},
	}

	for _, f := range flags {
		t.Run(f.name, func(t *testing.T) {
			pf := root.PersistentFlags().Lookup(f.name)
			if pf == nil {
				t.Fatalf("flag %q not found", f.name)
			}
			if pf.Shorthand != f.shorthand {
				t.Errorf("flag %q shorthand = %q, want %q", f.name, pf.Shorthand, f.shorthand)
			}
		})
	}
}
