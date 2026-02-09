package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---------- AtomicWrite tests ----------

func TestAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	data := []byte(`{"hello": "world"}`)

	if err := AtomicWrite(path, data, 0644); err != nil {
		t.Fatalf("AtomicWrite: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("content = %q, want %q", got, data)
	}

	// Verify no leftover temp files
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp") {
			t.Errorf("leftover temp file: %s", e.Name())
		}
	}
}

func TestAtomicWriteFailure(t *testing.T) {
	// Writing to a non-existent directory should fail
	path := filepath.Join(t.TempDir(), "no-such-dir", "test.json")

	err := AtomicWrite(path, []byte("data"), 0644)
	if err == nil {
		t.Fatal("expected error for non-existent directory")
	}

	// Verify no leftover temp files in the parent (which does exist)
	parentDir := filepath.Dir(filepath.Dir(path))
	entries, _ := os.ReadDir(parentDir)
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp") {
			t.Errorf("leftover temp file: %s", e.Name())
		}
	}
}

// ---------- Create tests ----------

func TestCreate(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "backups")

	subs := map[string]SubmoduleState{
		"plugins/auth": {SHA: "abc123", Branch: "develop"},
		"plugins/blog": {SHA: "def456", Branch: "main"},
	}

	filename, err := Create(dir, subs)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if !strings.HasPrefix(filename, "backup-") || !strings.HasSuffix(filename, ".json") {
		t.Errorf("unexpected filename format: %s", filename)
	}

	// Read back and verify structure
	fullPath := filepath.Join(dir, filename)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var b Backup
	if err := json.Unmarshal(data, &b); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if b.Version != 2 {
		t.Errorf("Version = %d, want 2", b.Version)
	}
	if len(b.Submodules) != 2 {
		t.Errorf("len(Submodules) = %d, want 2", len(b.Submodules))
	}
	if b.Submodules["plugins/auth"].SHA != "abc123" {
		t.Errorf("auth SHA = %q, want abc123", b.Submodules["plugins/auth"].SHA)
	}
	if b.Submodules["plugins/blog"].Branch != "main" {
		t.Errorf("blog Branch = %q, want main", b.Submodules["plugins/blog"].Branch)
	}
}

// ---------- Read tests ----------

func TestReadGoEra(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backup-20260209-103000.json")

	b := Backup{
		Version:   2,
		Timestamp: "2026-02-09T10:30:00Z",
		Submodules: map[string]SubmoduleState{
			"lib/core": {SHA: "aaa111", Branch: "develop"},
		},
	}
	data, _ := json.MarshalIndent(b, "", "  ")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.Version != 2 {
		t.Errorf("Version = %d, want 2", got.Version)
	}
	if got.Submodules["lib/core"].SHA != "aaa111" {
		t.Errorf("SHA = %q, want aaa111", got.Submodules["lib/core"].SHA)
	}
}

func TestReadBashEra(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".submodule-backup-20260209-103000.json")

	// Bash-era format: no version field
	raw := `{
  "timestamp": "2026-02-09T10:30:00+00:00",
  "submodules": {
    "plugins/auth": {"sha": "bbb222", "branch": "develop"},
    "plugins/blog": {"sha": "ccc333", "branch": "master"}
  }
}`
	if err := os.WriteFile(path, []byte(raw), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.Version != 1 {
		t.Errorf("Version = %d, want 1 (bash-era normalized)", got.Version)
	}
	if got.Submodules["plugins/auth"].SHA != "bbb222" {
		t.Errorf("auth SHA = %q, want bbb222", got.Submodules["plugins/auth"].SHA)
	}
	if got.Submodules["plugins/blog"].Branch != "master" {
		t.Errorf("blog Branch = %q, want master", got.Submodules["plugins/blog"].Branch)
	}
}

// ---------- List tests ----------

func TestList(t *testing.T) {
	root := t.TempDir()
	backupDir := filepath.Join(root, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create Go-era backups with different timestamps
	names := []string{
		"backup-20260201-100000.json",
		"backup-20260205-100000.json",
		"backup-20260209-100000.json",
	}
	for _, name := range names {
		path := filepath.Join(backupDir, name)
		b := Backup{Version: 2, Timestamp: "T", Submodules: map[string]SubmoduleState{}}
		data, _ := json.Marshal(b)
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatal(err)
		}
	}

	infos, err := List(backupDir)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(infos) != 3 {
		t.Fatalf("len = %d, want 3", len(infos))
	}

	// Verify sorted newest first
	if !strings.Contains(infos[0].Filename, "20260209") {
		t.Errorf("first entry should be newest, got %s", infos[0].Filename)
	}
	if !strings.Contains(infos[2].Filename, "20260201") {
		t.Errorf("last entry should be oldest, got %s", infos[2].Filename)
	}
}

func TestListWithBashEra(t *testing.T) {
	root := t.TempDir()
	backupDir := filepath.Join(root, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Go-era backup
	goPath := filepath.Join(backupDir, "backup-20260209-100000.json")
	data, _ := json.Marshal(Backup{Version: 2, Timestamp: "T", Submodules: map[string]SubmoduleState{}})
	if err := os.WriteFile(goPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Bash-era backup in parent directory
	bashPath := filepath.Join(root, ".submodule-backup-20260208-100000.json")
	bashData := `{"timestamp":"2026-02-08T10:00:00Z","submodules":{}}`
	if err := os.WriteFile(bashPath, []byte(bashData), 0644); err != nil {
		t.Fatal(err)
	}

	infos, err := List(backupDir)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(infos) != 2 {
		t.Fatalf("len = %d, want 2", len(infos))
	}

	// Go-era (Feb 9) should be first (newest)
	if infos[0].IsBashEra {
		t.Error("first entry should be Go-era (newest)")
	}
	// Bash-era (Feb 8) should be second
	if !infos[1].IsBashEra {
		t.Error("second entry should be bash-era")
	}
}

// ---------- Clean tests ----------

func TestCleanCountBased(t *testing.T) {
	root := t.TempDir()
	backupDir := filepath.Join(root, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create 10 Go-era backups
	for i := 1; i <= 10; i++ {
		name := fmt.Sprintf("backup-202602%02d-100000.json", i)
		path := filepath.Join(backupDir, name)
		data, _ := json.Marshal(Backup{Version: 2, Timestamp: "T", Submodules: map[string]SubmoduleState{}})
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatal(err)
		}
	}

	removed, err := Clean(backupDir, "3")
	if err != nil {
		t.Fatalf("Clean: %v", err)
	}
	if removed != 7 {
		t.Errorf("removed = %d, want 7", removed)
	}

	// Verify the 3 newest remain
	remaining, _ := List(backupDir)
	goEra := 0
	for _, info := range remaining {
		if !info.IsBashEra {
			goEra++
		}
	}
	if goEra != 3 {
		t.Errorf("remaining Go-era = %d, want 3", goEra)
	}
}

func TestCleanTimeBased(t *testing.T) {
	root := t.TempDir()
	backupDir := filepath.Join(root, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a recent backup (today)
	now := time.Now()
	recentName := fmt.Sprintf("backup-%s.json", now.Format("20060102-150405"))
	recentPath := filepath.Join(backupDir, recentName)
	data, _ := json.Marshal(Backup{Version: 2, Timestamp: now.Format(time.RFC3339), Submodules: map[string]SubmoduleState{}})
	if err := os.WriteFile(recentPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Create an old backup (30 days ago)
	old := now.AddDate(0, 0, -30)
	oldName := fmt.Sprintf("backup-%s.json", old.Format("20060102-150405"))
	oldPath := filepath.Join(backupDir, oldName)
	if err := os.WriteFile(oldPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	removed, err := Clean(backupDir, "7d")
	if err != nil {
		t.Fatalf("Clean: %v", err)
	}
	if removed != 1 {
		t.Errorf("removed = %d, want 1 (the old backup)", removed)
	}

	// Recent backup should remain
	remaining, _ := List(backupDir)
	goEra := 0
	for _, info := range remaining {
		if !info.IsBashEra {
			goEra++
		}
	}
	if goEra != 1 {
		t.Errorf("remaining Go-era = %d, want 1", goEra)
	}
}

func TestCleanDoesNotRemoveBashEra(t *testing.T) {
	root := t.TempDir()
	backupDir := filepath.Join(root, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Bash-era backup in parent
	bashPath := filepath.Join(root, ".submodule-backup-20240101-100000.json")
	if err := os.WriteFile(bashPath, []byte(`{"timestamp":"T","submodules":{}}`), 0644); err != nil {
		t.Fatal(err)
	}

	removed, err := Clean(backupDir, "1")
	if err != nil {
		t.Fatalf("Clean: %v", err)
	}
	if removed != 0 {
		t.Errorf("removed = %d, want 0 (bash-era should not be cleaned)", removed)
	}

	// Bash-era file should still exist
	if _, err := os.Stat(bashPath); os.IsNotExist(err) {
		t.Error("bash-era backup was removed but should not have been")
	}
}

// ---------- ParseKeepArg tests ----------

func TestParseKeepArg(t *testing.T) {
	tests := []struct {
		input     string
		wantMode  string
		wantValue int
		wantErr   bool
	}{
		{"5", "count", 5, false},
		{"7d", "time", 7, false},
		{"30d", "time", 30, false},
		{"1", "count", 1, false},
		{"0", "", 0, true},
		{"-1", "", 0, true},
		{"0d", "", 0, true},
		{"abc", "", 0, true},
		{"", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			mode, value, err := ParseKeepArg(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for %q", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.input, err)
			}
			if mode != tt.wantMode {
				t.Errorf("mode = %q, want %q", mode, tt.wantMode)
			}
			if value != tt.wantValue {
				t.Errorf("value = %d, want %d", value, tt.wantValue)
			}
		})
	}
}

// ---------- Compat tests ----------

func TestIsBashEraFilename(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{".submodule-backup-20260209-103000.json", true},
		{".submodule-backup-20240101-000000.json", true},
		{"backup-20260209-103000.json", false},
		{".submodule-backup-20260209.json", true}, // relaxed prefix/suffix match
		{"some-other-file.json", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsBashEraFilename(tt.name); got != tt.want {
				t.Errorf("IsBashEraFilename(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestReadBashEraFormat(t *testing.T) {
	data := []byte(`{
  "timestamp": "2026-02-09T10:30:00+00:00",
  "submodules": {
    "plugins/auth": {"sha": "abc123", "branch": "develop"}
  }
}`)

	b, err := ReadBashEra(data)
	if err != nil {
		t.Fatalf("ReadBashEra: %v", err)
	}
	if b.Version != 1 {
		t.Errorf("Version = %d, want 1", b.Version)
	}
	if b.Timestamp != "2026-02-09T10:30:00+00:00" {
		t.Errorf("Timestamp = %q", b.Timestamp)
	}
	if s, ok := b.Submodules["plugins/auth"]; !ok || s.SHA != "abc123" {
		t.Errorf("unexpected submodule state: %+v", b.Submodules)
	}
}

// ---------- ProjectName tests ----------

func TestProjectName(t *testing.T) {
	if got := ProjectName("/home/user/projects/myapp"); got != "myapp" {
		t.Errorf("ProjectName = %q, want myapp", got)
	}
}

// ---------- BackupDir tests ----------

func TestBackupDir(t *testing.T) {
	dir, err := BackupDir("myproject")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(dir, filepath.Join(".ssu", "myproject", "backups")) {
		t.Errorf("BackupDir = %q, expected to end with .ssu/myproject/backups", dir)
	}
}
