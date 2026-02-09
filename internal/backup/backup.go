package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// SubmoduleState holds the recorded state of a single submodule.
type SubmoduleState struct {
	SHA    string `json:"sha"`
	Branch string `json:"branch"`
}

// Backup is the Go-era backup format (version 2).
type Backup struct {
	Version    int                        `json:"version"`
	Timestamp  string                     `json:"timestamp"`
	Submodules map[string]SubmoduleState  `json:"submodules"`
}

// BackupInfo describes a backup file for listing purposes.
type BackupInfo struct {
	Filename  string
	Path      string
	Timestamp time.Time
	IsBashEra bool
}

// BackupDir returns the Go-era backup directory for a project:
// ~/.ssu/<projectName>/backups/
func BackupDir(projectName string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining home directory: %w", err)
	}
	return filepath.Join(home, ".ssu", projectName, "backups"), nil
}

// ProjectName returns the project name derived from the project root path.
// This matches the bash-era behavior: use the base directory name.
func ProjectName(projectRoot string) string {
	return filepath.Base(projectRoot)
}

// Create writes an atomic JSON backup file to backupDir and returns the filename.
func Create(backupDir string, submodules map[string]SubmoduleState) (string, error) {
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("creating backup directory: %w", err)
	}

	now := time.Now()
	b := Backup{
		Version:    2,
		Timestamp:  now.Format(time.RFC3339),
		Submodules: submodules,
	}

	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling backup: %w", err)
	}
	data = append(data, '\n')

	filename := fmt.Sprintf("backup-%s.json", now.Format("20060102-150405"))
	fullPath := filepath.Join(backupDir, filename)

	if err := AtomicWrite(fullPath, data, 0644); err != nil {
		return "", fmt.Errorf("writing backup: %w", err)
	}

	return filename, nil
}

// Read loads a backup file and returns a normalized Backup struct.
// It auto-detects Go-era (version field present) vs bash-era format.
func Read(path string) (*Backup, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Try Go-era format first
	var b Backup
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, fmt.Errorf("parsing backup %s: %w", filepath.Base(path), err)
	}

	// Version==0 means the version field was missing -- this is a bash-era backup
	if b.Version == 0 {
		return ReadBashEra(data)
	}

	return &b, nil
}

// List returns all backup files found in backupDir (Go-era) and its parent
// directory (bash-era), sorted by timestamp descending (newest first).
func List(backupDir string) ([]BackupInfo, error) {
	var infos []BackupInfo

	// Go-era backups: backupDir/*.json
	if entries, err := os.ReadDir(backupDir); err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			if strings.HasPrefix(e.Name(), "backup-") {
				ts := parseGoEraTimestamp(e.Name())
				infos = append(infos, BackupInfo{
					Filename:  e.Name(),
					Path:      filepath.Join(backupDir, e.Name()),
					Timestamp: ts,
					IsBashEra: false,
				})
			}
		}
	}

	// Bash-era backups: parent directory (backupDir/..)/.submodule-backup-*.json
	parentDir := filepath.Dir(backupDir)
	if entries, err := os.ReadDir(parentDir); err == nil {
		for _, e := range entries {
			if e.IsDir() || !IsBashEraFilename(e.Name()) {
				continue
			}
			ts := parseBashEraTimestamp(e.Name())
			infos = append(infos, BackupInfo{
				Filename:  e.Name(),
				Path:      filepath.Join(parentDir, e.Name()),
				Timestamp: ts,
				IsBashEra: true,
			})
		}
	}

	// Sort newest first
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Timestamp.After(infos[j].Timestamp)
	})

	return infos, nil
}

// Clean removes old backups from backupDir based on the keep argument.
// Only Go-era backups are cleaned (bash-era backups in the parent dir are not touched).
// Returns the count of removed files.
func Clean(backupDir string, keep string) (int, error) {
	mode, value, err := ParseKeepArg(keep)
	if err != nil {
		return 0, err
	}

	all, err := List(backupDir)
	if err != nil {
		return 0, err
	}

	// Filter to only Go-era backups (sorted newest first from List)
	var goEra []BackupInfo
	for _, info := range all {
		if !info.IsBashEra {
			goEra = append(goEra, info)
		}
	}

	var toRemove []BackupInfo
	switch mode {
	case "count":
		if len(goEra) > value {
			toRemove = goEra[value:]
		}
	case "time":
		cutoff := time.Now().AddDate(0, 0, -value)
		for _, info := range goEra {
			if info.Timestamp.Before(cutoff) {
				toRemove = append(toRemove, info)
			}
		}
	}

	removed := 0
	for _, info := range toRemove {
		if err := os.Remove(info.Path); err != nil {
			return removed, fmt.Errorf("removing %s: %w", info.Filename, err)
		}
		removed++
	}

	return removed, nil
}

// ParseKeepArg parses a keep argument string into a mode and value.
// Examples: "5" -> ("count", 5), "7d" -> ("time", 7).
// Returns error for non-positive values or unparseable strings.
func ParseKeepArg(arg string) (mode string, value int, err error) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return "", 0, fmt.Errorf("keep argument cannot be empty")
	}

	if strings.HasSuffix(arg, "d") {
		numStr := strings.TrimSuffix(arg, "d")
		n, parseErr := strconv.Atoi(numStr)
		if parseErr != nil {
			return "", 0, fmt.Errorf("invalid keep argument %q: %w", arg, parseErr)
		}
		if n <= 0 {
			return "", 0, fmt.Errorf("keep value must be positive, got %d", n)
		}
		return "time", n, nil
	}

	n, parseErr := strconv.Atoi(arg)
	if parseErr != nil {
		return "", 0, fmt.Errorf("invalid keep argument %q: %w", arg, parseErr)
	}
	if n <= 0 {
		return "", 0, fmt.Errorf("keep value must be positive, got %d", n)
	}
	return "count", n, nil
}

// parseGoEraTimestamp extracts a timestamp from a Go-era backup filename.
// Format: backup-YYYYMMDD-HHMMSS.json
func parseGoEraTimestamp(name string) time.Time {
	// Remove prefix "backup-" and suffix ".json"
	s := strings.TrimPrefix(name, "backup-")
	s = strings.TrimSuffix(s, ".json")
	t, err := time.Parse("20060102-150405", s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// parseBashEraTimestamp extracts a timestamp from a bash-era backup filename.
// Format: .submodule-backup-YYYYMMDD-HHMMSS.json
func parseBashEraTimestamp(name string) time.Time {
	s := strings.TrimPrefix(name, ".submodule-backup-")
	s = strings.TrimSuffix(s, ".json")
	t, err := time.Parse("20060102-150405", s)
	if err != nil {
		return time.Time{}
	}
	return t
}
