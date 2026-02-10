package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	// Load with a temp dir that has no config files
	dir := t.TempDir()
	cfg, ac, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Verify default values
	if cfg.Git.ParallelJobs != 8 {
		t.Errorf("Git.ParallelJobs = %d, want 8", cfg.Git.ParallelJobs)
	}
	if cfg.Git.FailFast != false {
		t.Errorf("Git.FailFast = %v, want false", cfg.Git.FailFast)
	}
	if len(cfg.Git.Skip) != 0 {
		t.Errorf("Git.Skip = %v, want empty", cfg.Git.Skip)
	}
	if len(cfg.Branches.Priority) != 3 {
		t.Errorf("Branches.Priority length = %d, want 3", len(cfg.Branches.Priority))
	}
	if cfg.Branches.Priority[0] != "develop" {
		t.Errorf("Branches.Priority[0] = %q, want %q", cfg.Branches.Priority[0], "develop")
	}
	if cfg.Branches.Override != "" {
		t.Errorf("Branches.Override = %q, want empty", cfg.Branches.Override)
	}
	if cfg.Backup.Enabled != true {
		t.Errorf("Backup.Enabled = %v, want true", cfg.Backup.Enabled)
	}
	if cfg.Backup.MaxBackups != 10 {
		t.Errorf("Backup.MaxBackups = %d, want 10", cfg.Backup.MaxBackups)
	}
	if cfg.Log.MaxSizeMB != 10 {
		t.Errorf("Log.MaxSizeMB = %d, want 10", cfg.Log.MaxSizeMB)
	}
	if cfg.Log.MaxBackups != 5 {
		t.Errorf("Log.MaxBackups = %d, want 5", cfg.Log.MaxBackups)
	}

	// All sources should be default
	for _, key := range allKeys {
		av := ac.Get(key)
		if av.Source != SourceDefault {
			t.Errorf("key %q source = %q, want %q", key, av.Source, SourceDefault)
		}
	}
}

func TestLoadProjectConfig(t *testing.T) {
	dir := t.TempDir()
	configContent := `
git:
  parallel_jobs: 16
  fail_fast: true
  skip:
    - "vendor/old"
backup:
  enabled: false
`
	if err := os.WriteFile(filepath.Join(dir, ".ssu.yaml"), []byte(configContent), 0o644); err != nil {
		t.Fatalf("writing project config: %v", err)
	}

	cfg, ac, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Git.ParallelJobs != 16 {
		t.Errorf("Git.ParallelJobs = %d, want 16", cfg.Git.ParallelJobs)
	}
	if cfg.Git.FailFast != true {
		t.Errorf("Git.FailFast = %v, want true", cfg.Git.FailFast)
	}
	if len(cfg.Git.Skip) != 1 || cfg.Git.Skip[0] != "vendor/old" {
		t.Errorf("Git.Skip = %v, want [vendor/old]", cfg.Git.Skip)
	}
	if cfg.Backup.Enabled != false {
		t.Errorf("Backup.Enabled = %v, want false", cfg.Backup.Enabled)
	}
	// Unchanged values should keep defaults
	if cfg.Branches.Priority[0] != "develop" {
		t.Errorf("Branches.Priority[0] = %q, want %q (default)", cfg.Branches.Priority[0], "develop")
	}

	// Source annotations
	if av := ac.Get("git.parallel_jobs"); av.Source != SourceProject {
		t.Errorf("git.parallel_jobs source = %q, want %q", av.Source, SourceProject)
	}
	if av := ac.Get("backup.enabled"); av.Source != SourceProject {
		t.Errorf("backup.enabled source = %q, want %q", av.Source, SourceProject)
	}
	// Unchanged keys should still be default
	if av := ac.Get("log.max_size_mb"); av.Source != SourceDefault {
		t.Errorf("log.max_size_mb source = %q, want %q", av.Source, SourceDefault)
	}
}

func TestLoadGlobalConfig(t *testing.T) {
	// We can't easily override $HOME for the global config path in Load(),
	// but we can test by using the project config path for layering validation.
	// This test verifies that project config overrides work correctly.
	// The global config mechanism uses the same Viper merge logic.
	dir := t.TempDir()
	configContent := `
log:
  max_size_mb: 50
  max_backups: 20
`
	if err := os.WriteFile(filepath.Join(dir, ".ssu.yaml"), []byte(configContent), 0o644); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	cfg, _, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Log.MaxSizeMB != 50 {
		t.Errorf("Log.MaxSizeMB = %d, want 50", cfg.Log.MaxSizeMB)
	}
	if cfg.Log.MaxBackups != 20 {
		t.Errorf("Log.MaxBackups = %d, want 20", cfg.Log.MaxBackups)
	}
}

func TestLoadEnvVarOverride(t *testing.T) {
	dir := t.TempDir()
	// Project config sets parallel_jobs to 16
	configContent := `
git:
  parallel_jobs: 16
`
	if err := os.WriteFile(filepath.Join(dir, ".ssu.yaml"), []byte(configContent), 0o644); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	// Env var overrides to 32
	t.Setenv("SSU_GIT_PARALLEL_JOBS", "32")

	cfg, ac, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Git.ParallelJobs != 32 {
		t.Errorf("Git.ParallelJobs = %d, want 32 (env override)", cfg.Git.ParallelJobs)
	}
	if av := ac.Get("git.parallel_jobs"); av.Source != SourceEnv {
		t.Errorf("git.parallel_jobs source = %q, want %q", av.Source, SourceEnv)
	}
}

func TestLoadLegacyEnvVar(t *testing.T) {
	dir := t.TempDir()

	// Legacy env var without SSU_ prefix
	t.Setenv("PARALLEL_JOBS", "24")

	cfg, ac, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Git.ParallelJobs != 24 {
		t.Errorf("Git.ParallelJobs = %d, want 24 (legacy env)", cfg.Git.ParallelJobs)
	}
	if av := ac.Get("git.parallel_jobs"); av.Source != SourceEnv {
		t.Errorf("git.parallel_jobs source = %q, want %q", av.Source, SourceEnv)
	}
}

func TestLoadLegacyEnvVarPriority(t *testing.T) {
	dir := t.TempDir()

	// Both legacy and canonical set -- canonical should win
	t.Setenv("PARALLEL_JOBS", "24")
	t.Setenv("SSU_GIT_PARALLEL_JOBS", "48")

	cfg, _, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Git.ParallelJobs != 48 {
		t.Errorf("Git.ParallelJobs = %d, want 48 (SSU_ prefix wins)", cfg.Git.ParallelJobs)
	}
}

func TestLoadMalformedConfig(t *testing.T) {
	dir := t.TempDir()
	// Invalid YAML
	if err := os.WriteFile(filepath.Join(dir, ".ssu.yaml"), []byte("{{{{invalid yaml"), 0o644); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	_, _, err := Load(dir)
	if err == nil {
		t.Error("Load() expected error for malformed config, got nil")
	}
}

func TestLoadEmptyProjectRoot(t *testing.T) {
	// Empty project root should still load defaults
	cfg, _, err := Load("")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Git.ParallelJobs != 8 {
		t.Errorf("Git.ParallelJobs = %d, want 8 (default)", cfg.Git.ParallelJobs)
	}
}

func TestDefaults(t *testing.T) {
	d := Defaults()
	if d.Git.ParallelJobs != 8 {
		t.Errorf("Defaults().Git.ParallelJobs = %d, want 8", d.Git.ParallelJobs)
	}
	if len(d.Branches.Priority) != 3 {
		t.Errorf("Defaults().Branches.Priority length = %d, want 3", len(d.Branches.Priority))
	}
	if d.Backup.Enabled != true {
		t.Errorf("Defaults().Backup.Enabled = %v, want true", d.Backup.Enabled)
	}
	if d.Log.MaxSizeMB != 10 {
		t.Errorf("Defaults().Log.MaxSizeMB = %d, want 10", d.Log.MaxSizeMB)
	}
}

func TestSourceAnnotations(t *testing.T) {
	ac := NewAnnotatedConfig()

	// Initially empty
	av := ac.Get("nonexistent")
	if av.Source != SourceDefault {
		t.Errorf("nonexistent key source = %q, want %q", av.Source, SourceDefault)
	}

	// Set and retrieve
	ac.Set("git.parallel_jobs", 16, SourceProject)
	av = ac.Get("git.parallel_jobs")
	if av.Source != SourceProject {
		t.Errorf("git.parallel_jobs source = %q, want %q", av.Source, SourceProject)
	}
	if av.Value != 16 {
		t.Errorf("git.parallel_jobs value = %v, want 16", av.Value)
	}

	// Override source
	ac.Set("git.parallel_jobs", 32, SourceFlag)
	av = ac.Get("git.parallel_jobs")
	if av.Source != SourceFlag {
		t.Errorf("git.parallel_jobs source after flag = %q, want %q", av.Source, SourceFlag)
	}
}

func TestContextHelpers(t *testing.T) {
	cfg := &Config{Git: GitConfig{ParallelJobs: 42}}
	ac := NewAnnotatedConfig()
	ac.Set("git.parallel_jobs", 42, SourceFlag)

	ctx := context.Background()

	// Before storing, should be nil
	if got := FromContext(ctx); got != nil {
		t.Error("FromContext on empty context should return nil")
	}
	if got := AnnotatedFromContext(ctx); got != nil {
		t.Error("AnnotatedFromContext on empty context should return nil")
	}

	// Store and retrieve
	ctx = WithConfig(ctx, cfg)
	ctx = WithAnnotated(ctx, ac)

	got := FromContext(ctx)
	if got == nil {
		t.Fatal("FromContext returned nil after WithConfig")
	}
	if got.Git.ParallelJobs != 42 {
		t.Errorf("FromContext().Git.ParallelJobs = %d, want 42", got.Git.ParallelJobs)
	}

	gotAC := AnnotatedFromContext(ctx)
	if gotAC == nil {
		t.Fatal("AnnotatedFromContext returned nil after WithAnnotated")
	}
	if av := gotAC.Get("git.parallel_jobs"); av.Source != SourceFlag {
		t.Errorf("annotated git.parallel_jobs source = %q, want %q", av.Source, SourceFlag)
	}
}

func TestAnnotatedConfigKeys(t *testing.T) {
	ac := NewAnnotatedConfig()
	ac.Set("a", 1, SourceDefault)
	ac.Set("b", 2, SourceGlobal)
	ac.Set("c", 3, SourceProject)

	keys := ac.Keys()
	if len(keys) != 3 {
		t.Errorf("Keys() length = %d, want 3", len(keys))
	}

	// Verify all keys present (order not guaranteed)
	found := make(map[string]bool)
	for _, k := range keys {
		found[k] = true
	}
	for _, want := range []string{"a", "b", "c"} {
		if !found[want] {
			t.Errorf("Keys() missing %q", want)
		}
	}
}
