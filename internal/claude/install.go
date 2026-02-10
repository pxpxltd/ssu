package claude

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// InstallResult reports what happened during installation.
type InstallResult struct {
	Installed []string // files written (new or overwritten)
	Skipped   []string // files identical to existing
	Dir       string   // target directory
}

// InstallCommands copies embedded slash command files to ~/.claude/commands/ssu/.
// If force is false, it returns an error when an existing file differs from the
// embedded version. Identical files are silently skipped regardless of force.
func InstallCommands(force bool) (*InstallResult, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine home directory: %w", err)
	}
	return InstallCommandsTo(filepath.Join(home, ".claude", "commands", "ssu"), force)
}

// InstallCommandsTo copies embedded command files to a specific directory.
// Exported for testing (callers pass a temp dir).
func InstallCommandsTo(targetDir string, force bool) (*InstallResult, error) {
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return nil, fmt.Errorf("cannot create directory %s: %w", targetDir, err)
	}

	result := &InstallResult{Dir: targetDir}

	err := fs.WalkDir(CommandsFS, "commands", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		content, readErr := CommandsFS.ReadFile(path)
		if readErr != nil {
			return readErr
		}

		name := filepath.Base(path)
		dest := filepath.Join(targetDir, name)

		// Check existing file
		if existing, readErr := os.ReadFile(dest); readErr == nil {
			if string(existing) == string(content) {
				result.Skipped = append(result.Skipped, name)
				return nil
			}
			if !force {
				return fmt.Errorf("file %s already exists and differs (use --force to overwrite)", dest)
			}
		}

		if writeErr := os.WriteFile(dest, content, 0o644); writeErr != nil {
			return writeErr
		}
		result.Installed = append(result.Installed, name)
		return nil
	})

	return result, err
}
