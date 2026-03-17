package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallCommandsTo_Fresh(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "commands", "ssu")

	result, err := InstallCommandsTo(dir, false)
	if err != nil {
		t.Fatalf("InstallCommandsTo: %v", err)
	}

	if len(result.Installed) != 9 {
		t.Errorf("Installed = %d, want 9; got %v", len(result.Installed), result.Installed)
	}
	if len(result.Skipped) != 0 {
		t.Errorf("Skipped = %d, want 0", len(result.Skipped))
	}
	if result.Dir != dir {
		t.Errorf("Dir = %q, want %q", result.Dir, dir)
	}

	// Verify all nine files exist on disk
	for _, name := range []string{"status.md", "update.md", "push.md", "checkout.md", "project.md", "exec.md", "rollback.md", "rm.md", "allow.md"} {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", path)
		}
	}
}

func TestInstallCommandsTo_SkipIdentical(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "ssu")

	// First install
	_, err := InstallCommandsTo(dir, false)
	if err != nil {
		t.Fatalf("first install: %v", err)
	}

	// Second install -- all files identical
	result, err := InstallCommandsTo(dir, false)
	if err != nil {
		t.Fatalf("second install: %v", err)
	}

	if len(result.Skipped) != 9 {
		t.Errorf("Skipped = %d, want 9; got %v", len(result.Skipped), result.Skipped)
	}
	if len(result.Installed) != 0 {
		t.Errorf("Installed = %d, want 0; got %v", len(result.Installed), result.Installed)
	}
}

func TestInstallCommandsTo_ErrorOnDifferent(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "ssu")

	// First install
	_, err := InstallCommandsTo(dir, false)
	if err != nil {
		t.Fatalf("first install: %v", err)
	}

	// Modify one file
	statusPath := filepath.Join(dir, "status.md")
	if err := os.WriteFile(statusPath, []byte("user-customized content"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Second install without force -- should error
	_, err = InstallCommandsTo(dir, false)
	if err == nil {
		t.Fatal("expected error for differing file")
	}
	if !strings.Contains(err.Error(), "already exists and differs") {
		t.Errorf("error = %q, want it to contain 'already exists and differs'", err.Error())
	}
}

func TestInstallCommandsTo_ForceOverwrite(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "ssu")

	// First install
	_, err := InstallCommandsTo(dir, false)
	if err != nil {
		t.Fatalf("first install: %v", err)
	}

	// Modify one file
	updatePath := filepath.Join(dir, "update.md")
	if err := os.WriteFile(updatePath, []byte("user-customized content"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Second install with force -- should succeed
	result, err := InstallCommandsTo(dir, true)
	if err != nil {
		t.Fatalf("force install: %v", err)
	}

	// The modified file should be in Installed
	found := false
	for _, name := range result.Installed {
		if name == "update.md" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected update.md in Installed, got %v", result.Installed)
	}

	// The other eight identical files should be in Skipped
	if len(result.Skipped) != 8 {
		t.Errorf("Skipped = %d, want 8; got %v", len(result.Skipped), result.Skipped)
	}
}

func TestInstallCommandsTo_CreatesDirectory(t *testing.T) {
	// Use a deeply nested path that doesn't exist
	dir := filepath.Join(t.TempDir(), "a", "b", "c", "ssu")

	result, err := InstallCommandsTo(dir, false)
	if err != nil {
		t.Fatalf("InstallCommandsTo: %v", err)
	}

	if len(result.Installed) != 9 {
		t.Errorf("Installed = %d, want 9", len(result.Installed))
	}

	// Verify directory was created and files exist
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 9 {
		t.Errorf("directory entries = %d, want 9", len(entries))
	}
}

func TestSnippetContent(t *testing.T) {
	if SnippetContent == "" {
		t.Fatal("SnippetContent is empty")
	}
	if !strings.Contains(SnippetContent, "ssu") {
		t.Error("SnippetContent does not contain 'ssu'")
	}
	if !strings.Contains(SnippetContent, "ssu update --auto") {
		t.Error("SnippetContent does not contain 'ssu update --auto'")
	}
}
