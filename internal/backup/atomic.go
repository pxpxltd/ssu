// Package backup implements atomic JSON backup creation, bash-era format
// compatibility, backup management (list/clean), and rollback for SSU.
package backup

import (
	"os"
	"path/filepath"
)

// AtomicWrite writes data to path atomically using temp-file + fsync + rename.
// The temp file is created in the same directory as path to ensure same-device
// rename, which is atomic on POSIX systems.
func AtomicWrite(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)

	tmp, err := os.CreateTemp(dir, ".ssu-backup-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	// Cleanup: remove temp file on any error (tmpPath is cleared after successful rename)
	defer func() {
		if tmpPath != "" {
			os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, perm); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}

	tmpPath = "" // success: skip cleanup
	return nil
}
