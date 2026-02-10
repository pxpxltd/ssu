package backup

import (
	"encoding/json"
	"strings"
)

// bashEraBackup matches the JSON format produced by the bash-era ssu script.
// It has no "version" field and uses a flat submodules map.
type bashEraBackup struct {
	Timestamp  string                       `json:"timestamp"`
	Submodules map[string]bashEraSubmodule  `json:"submodules"`
}

// bashEraSubmodule matches the per-submodule state in bash-era backups.
type bashEraSubmodule struct {
	SHA    string `json:"sha"`
	Branch string `json:"branch"`
}

// ReadBashEra parses a bash-era backup file (no version field) and normalizes
// it into the Go-era Backup struct with Version=1.
func ReadBashEra(data []byte) (*Backup, error) {
	var raw bashEraBackup
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	subs := make(map[string]SubmoduleState, len(raw.Submodules))
	for path, s := range raw.Submodules {
		subs[path] = SubmoduleState{
			SHA:    s.SHA,
			Branch: s.Branch,
		}
	}

	return &Backup{
		Version:    1,
		Timestamp:  raw.Timestamp,
		Submodules: subs,
	}, nil
}

// IsBashEraFilename returns true if name matches the bash-era backup pattern:
// .submodule-backup-YYYYMMDD-HHMMSS.json (starts with dot).
func IsBashEraFilename(name string) bool {
	return strings.HasPrefix(name, ".submodule-backup-") && strings.HasSuffix(name, ".json")
}
