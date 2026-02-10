package logging

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

// MultiHandler fans out log records to multiple slog.Handler implementations.
type MultiHandler struct {
	handlers []slog.Handler
}

// Enabled reports whether any of the underlying handlers is enabled for the level.
func (m *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

// Handle writes the record to every enabled handler.
func (m *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range m.handlers {
		if h.Enabled(ctx, r.Level) {
			if err := h.Handle(ctx, r); err != nil {
				return err
			}
		}
	}
	return nil
}

// WithAttrs returns a new MultiHandler where each underlying handler has the attrs.
func (m *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}
	return &MultiHandler{handlers: handlers}
}

// WithGroup returns a new MultiHandler where each underlying handler has the group.
func (m *MultiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithGroup(name)
	}
	return &MultiHandler{handlers: handlers}
}

// InitLogger creates a slog.Logger that writes to a rotating log file.
//
// The log file is created at logDir/submodule-update.log and rotated by
// lumberjack when it exceeds maxSizeMB megabytes, keeping up to maxBackups
// old files.
//
// When verbose is true, a second handler writes DEBUG+ output to stderr.
// When verbose is false, only INFO+ goes to the file and nothing to stderr.
func InitLogger(logDir string, verbose bool, maxSizeMB, maxBackups int) (*slog.Logger, error) {
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, err
	}

	lj := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, "submodule-update.log"),
		MaxSize:    maxSizeMB,
		MaxBackups: maxBackups,
		LocalTime:  true,
	}

	fileHandler := NewBracketHandler(lj, slog.LevelInfo)

	if verbose {
		stderrHandler := NewBracketHandler(os.Stderr, slog.LevelDebug)
		multi := &MultiHandler{handlers: []slog.Handler{fileHandler, stderrHandler}}
		return slog.New(multi), nil
	}

	return slog.New(fileHandler), nil
}

// LogDir returns the standard log directory path for a project:
// ~/.ssu/<projectName>/logs
func LogDir(projectName string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback: use current directory
		home = "."
	}
	return filepath.Join(home, ".ssu", projectName, "logs")
}
