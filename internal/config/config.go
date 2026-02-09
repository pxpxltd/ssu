// Package config implements layered YAML configuration for SSU.
//
// Configuration is loaded in priority order (highest wins):
//
//  1. CLI flags (--jobs, --verbose, etc.)
//  2. Environment variables (SSU_GIT_PARALLEL_JOBS, etc.)
//  3. Project config (.ssu.yaml in project root)
//  4. Global config (~/.ssu/config.yaml)
//  5. Built-in defaults
//
// The legacy PARALLEL_JOBS env var (without SSU_ prefix) is silently supported
// but SSU_GIT_PARALLEL_JOBS takes priority when both are set.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all SSU configuration values.
type Config struct {
	Git      GitConfig    `mapstructure:"git"`
	Branches BranchConfig `mapstructure:"branches"`
	Backup   BackupConfig `mapstructure:"backup"`
	Log      LogConfig    `mapstructure:"log"`
}

// GitConfig holds git operation settings.
type GitConfig struct {
	ParallelJobs int      `mapstructure:"parallel_jobs"`
	Skip         []string `mapstructure:"skip"`
	FailFast     bool     `mapstructure:"fail_fast"`
}

// BranchConfig holds branch detection settings.
type BranchConfig struct {
	Priority []string `mapstructure:"priority"`
	Override string   `mapstructure:"override"`
}

// BackupConfig holds backup behavior settings.
type BackupConfig struct {
	Enabled    bool `mapstructure:"enabled"`
	MaxBackups int  `mapstructure:"max_backups"`
}

// LogConfig holds logging settings.
type LogConfig struct {
	MaxSizeMB  int `mapstructure:"max_size_mb"`
	MaxBackups int `mapstructure:"max_backups"`
}

// allKeys lists every config key in dot-notation for annotation tracking.
var allKeys = []string{
	"git.parallel_jobs",
	"git.skip",
	"git.fail_fast",
	"branches.priority",
	"branches.override",
	"backup.enabled",
	"backup.max_backups",
	"log.max_size_mb",
	"log.max_backups",
}

// Defaults returns a Config populated with built-in default values.
func Defaults() Config {
	return Config{
		Git: GitConfig{
			ParallelJobs: 8,
			FailFast:     false,
		},
		Branches: BranchConfig{
			Priority: []string{"develop", "master", "main"},
		},
		Backup: BackupConfig{
			Enabled:    true,
			MaxBackups: 10,
		},
		Log: LogConfig{
			MaxSizeMB:  10,
			MaxBackups: 5,
		},
	}
}

// Load reads configuration from defaults, global config, project config,
// and environment variables. It returns the merged Config and an AnnotatedConfig
// tracking which layer set each value.
//
// The projectRoot parameter is the git repository root (or cwd if not in a repo).
// Missing config files are silently skipped. Malformed config files return an error.
func Load(projectRoot string) (*Config, *AnnotatedConfig, error) {
	v := viper.New()

	// 1. Set defaults
	v.SetDefault("git.parallel_jobs", 8)
	v.SetDefault("git.skip", []string{})
	v.SetDefault("git.fail_fast", false)
	v.SetDefault("branches.priority", []string{"develop", "master", "main"})
	v.SetDefault("branches.override", "")
	v.SetDefault("backup.enabled", true)
	v.SetDefault("backup.max_backups", 10)
	v.SetDefault("log.max_size_mb", 10)
	v.SetDefault("log.max_backups", 5)

	ac := NewAnnotatedConfig()
	for _, key := range allKeys {
		ac.Set(key, v.Get(key), SourceDefault)
	}

	// 2. Read global config: ~/.ssu/config.yaml
	globalRead := false
	home, err := os.UserHomeDir()
	if err == nil {
		globalPath := filepath.Join(home, ".ssu", "config.yaml")
		if _, statErr := os.Stat(globalPath); statErr == nil {
			gv := viper.New()
			gv.SetConfigFile(globalPath)
			if readErr := gv.ReadInConfig(); readErr != nil {
				return nil, nil, fmt.Errorf("reading global config %s: %w", globalPath, readErr)
			}
			if mergeErr := v.MergeConfigMap(gv.AllSettings()); mergeErr != nil {
				return nil, nil, fmt.Errorf("merging global config: %w", mergeErr)
			}
			globalRead = true
			// Update annotations for keys that changed
			for _, key := range allKeys {
				if gv.IsSet(key) {
					ac.Set(key, v.Get(key), SourceGlobal)
				}
			}
		}
	}
	_ = globalRead

	// 3. Read project config: {projectRoot}/.ssu.yaml
	if projectRoot != "" {
		projectPath := filepath.Join(projectRoot, ".ssu.yaml")
		if _, statErr := os.Stat(projectPath); statErr == nil {
			pv := viper.New()
			pv.SetConfigFile(projectPath)
			if readErr := pv.ReadInConfig(); readErr != nil {
				return nil, nil, fmt.Errorf("reading project config %s: %w", projectPath, readErr)
			}
			if mergeErr := v.MergeConfigMap(pv.AllSettings()); mergeErr != nil {
				return nil, nil, fmt.Errorf("merging project config: %w", mergeErr)
			}
			// Update annotations for keys that changed
			for _, key := range allKeys {
				if pv.IsSet(key) {
					ac.Set(key, v.Get(key), SourceProject)
				}
			}
		}
	}

	// 4. Environment variables: SSU_ prefix
	v.SetEnvPrefix("SSU")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Check which env vars are actually set and update annotations
	envMappings := map[string]string{
		"git.parallel_jobs": "SSU_GIT_PARALLEL_JOBS",
		"git.fail_fast":     "SSU_GIT_FAIL_FAST",
		"branches.override":  "SSU_BRANCHES_OVERRIDE",
		"backup.enabled":     "SSU_BACKUP_ENABLED",
		"backup.max_backups": "SSU_BACKUP_MAX_BACKUPS",
		"log.max_size_mb":    "SSU_LOG_MAX_SIZE_MB",
		"log.max_backups":    "SSU_LOG_MAX_BACKUPS",
	}
	for key, envVar := range envMappings {
		if val := os.Getenv(envVar); val != "" {
			ac.Set(key, v.Get(key), SourceEnv)
		}
	}

	// 5. Legacy env var: PARALLEL_JOBS (without SSU_ prefix)
	// Only applies if SSU_GIT_PARALLEL_JOBS is not set.
	if legacy := os.Getenv("PARALLEL_JOBS"); legacy != "" {
		if os.Getenv("SSU_GIT_PARALLEL_JOBS") == "" {
			if n, parseErr := strconv.Atoi(legacy); parseErr == nil {
				v.Set("git.parallel_jobs", n)
				ac.Set("git.parallel_jobs", n, SourceEnv)
			}
		}
	}

	// Unmarshal into Config struct
	var cfg Config
	if unmarshalErr := v.Unmarshal(&cfg); unmarshalErr != nil {
		return nil, nil, fmt.Errorf("parsing config: %w", unmarshalErr)
	}

	return &cfg, ac, nil
}
