package logging

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestBracketHandler_Format(t *testing.T) {
	var buf bytes.Buffer
	h := NewBracketHandler(&buf, slog.LevelInfo)

	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	r := slog.NewRecord(ts, slog.LevelInfo, "Starting scan", 0)

	if err := h.Handle(context.Background(), r); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	got := buf.String()
	want := "[2024-01-15 10:30:00] [INFO] Starting scan\n"
	if got != want {
		t.Errorf("format mismatch:\n  got:  %q\n  want: %q", got, want)
	}
}

func TestBracketHandler_FormatRegex(t *testing.T) {
	var buf bytes.Buffer
	h := NewBracketHandler(&buf, slog.LevelInfo)

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)
	if err := h.Handle(context.Background(), r); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	// Match pattern: [YYYY-MM-DD HH:MM:SS] [INFO] message
	pattern := `^\[\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\] \[INFO\] test message\n$`
	if !regexp.MustCompile(pattern).MatchString(buf.String()) {
		t.Errorf("output does not match pattern %s:\n  got: %q", pattern, buf.String())
	}
}

func TestBracketHandler_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	h := NewBracketHandler(&buf, slog.LevelInfo)

	// DEBUG should be filtered out
	if h.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("DEBUG should not be enabled when level is INFO")
	}

	// INFO should be enabled
	if !h.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("INFO should be enabled when level is INFO")
	}

	// WARN should be enabled
	if !h.Enabled(context.Background(), slog.LevelWarn) {
		t.Error("WARN should be enabled when level is INFO")
	}

	// ERROR should be enabled
	if !h.Enabled(context.Background(), slog.LevelError) {
		t.Error("ERROR should be enabled when level is INFO")
	}
}

func TestBracketHandler_WarnLevel(t *testing.T) {
	var buf bytes.Buffer
	h := NewBracketHandler(&buf, slog.LevelInfo)

	ts := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	r := slog.NewRecord(ts, slog.LevelWarn, "something fishy", 0)

	if err := h.Handle(context.Background(), r); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, "[WARN]") {
		t.Errorf("expected [WARN] in output, got: %q", got)
	}
	// Must NOT contain [WARNING]
	if strings.Contains(got, "[WARNING]") {
		t.Errorf("expected [WARN] not [WARNING], got: %q", got)
	}
}

func TestBracketHandler_ErrorLevel(t *testing.T) {
	var buf bytes.Buffer
	h := NewBracketHandler(&buf, slog.LevelInfo)

	ts := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	r := slog.NewRecord(ts, slog.LevelError, "something broke", 0)

	if err := h.Handle(context.Background(), r); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	want := "[2024-06-01 12:00:00] [ERROR] something broke\n"
	if buf.String() != want {
		t.Errorf("format mismatch:\n  got:  %q\n  want: %q", buf.String(), want)
	}
}

func TestBracketHandler_DebugLevel(t *testing.T) {
	var buf bytes.Buffer
	h := NewBracketHandler(&buf, slog.LevelDebug)

	ts := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	r := slog.NewRecord(ts, slog.LevelDebug, "debug info", 0)

	if err := h.Handle(context.Background(), r); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	want := "[2024-06-01 12:00:00] [DEBUG] debug info\n"
	if buf.String() != want {
		t.Errorf("format mismatch:\n  got:  %q\n  want: %q", buf.String(), want)
	}
}

func TestInitLogger_CreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "logs")

	logger, err := InitLogger(dir, false, 10, 5)
	if err != nil {
		t.Fatalf("InitLogger returned error: %v", err)
	}
	if logger == nil {
		t.Fatal("InitLogger returned nil logger")
	}

	// Directory should exist
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("log directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("log path is not a directory")
	}
}

func TestInitLogger_CreatesLogFile(t *testing.T) {
	dir := t.TempDir()

	logger, err := InitLogger(dir, false, 10, 5)
	if err != nil {
		t.Fatalf("InitLogger returned error: %v", err)
	}

	// Write a log entry to trigger file creation
	logger.Info("test entry")

	logFile := filepath.Join(dir, "submodule-update.log")
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	got := string(data)
	if !strings.Contains(got, "[INFO] test entry") {
		t.Errorf("log file content unexpected: %q", got)
	}
}

func TestInitLogger_NonVerbose_NoStderr(t *testing.T) {
	dir := t.TempDir()

	logger, err := InitLogger(dir, false, 10, 5)
	if err != nil {
		t.Fatalf("InitLogger returned error: %v", err)
	}

	// Non-verbose logger should only have a file handler (not multi).
	// We verify by checking the handler type.
	handler := logger.Handler()
	if _, ok := handler.(*MultiHandler); ok {
		t.Error("non-verbose logger should not use MultiHandler")
	}
	if _, ok := handler.(*BracketHandler); !ok {
		t.Error("non-verbose logger should use BracketHandler directly")
	}
}

func TestInitLogger_Verbose_MultiHandler(t *testing.T) {
	dir := t.TempDir()

	logger, err := InitLogger(dir, true, 10, 5)
	if err != nil {
		t.Fatalf("InitLogger returned error: %v", err)
	}

	handler := logger.Handler()
	multi, ok := handler.(*MultiHandler)
	if !ok {
		t.Fatal("verbose logger should use MultiHandler")
	}
	if len(multi.handlers) != 2 {
		t.Errorf("expected 2 handlers (file + stderr), got %d", len(multi.handlers))
	}
}

func TestMultiHandler_FanOut(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	h1 := NewBracketHandler(&buf1, slog.LevelInfo)
	h2 := NewBracketHandler(&buf2, slog.LevelInfo)

	multi := &MultiHandler{handlers: []slog.Handler{h1, h2}}
	logger := slog.New(multi)

	logger.Info("fan out test")

	if !strings.Contains(buf1.String(), "fan out test") {
		t.Errorf("handler 1 did not receive message: %q", buf1.String())
	}
	if !strings.Contains(buf2.String(), "fan out test") {
		t.Errorf("handler 2 did not receive message: %q", buf2.String())
	}
}

func TestMultiHandler_LevelRouting(t *testing.T) {
	var infoBuf, debugBuf bytes.Buffer
	infoHandler := NewBracketHandler(&infoBuf, slog.LevelInfo)
	debugHandler := NewBracketHandler(&debugBuf, slog.LevelDebug)

	multi := &MultiHandler{handlers: []slog.Handler{infoHandler, debugHandler}}
	logger := slog.New(multi)

	logger.Debug("debug only")
	logger.Info("info for both")

	// Debug handler should have both messages
	if !strings.Contains(debugBuf.String(), "debug only") {
		t.Error("debug handler should receive DEBUG message")
	}
	if !strings.Contains(debugBuf.String(), "info for both") {
		t.Error("debug handler should receive INFO message")
	}

	// Info handler should only have the INFO message
	if strings.Contains(infoBuf.String(), "debug only") {
		t.Error("info handler should NOT receive DEBUG message")
	}
	if !strings.Contains(infoBuf.String(), "info for both") {
		t.Error("info handler should receive INFO message")
	}
}

func TestMultiHandler_Enabled(t *testing.T) {
	var buf bytes.Buffer
	h := NewBracketHandler(&buf, slog.LevelWarn)

	multi := &MultiHandler{handlers: []slog.Handler{h}}

	// WARN is enabled (matches handler level)
	if !multi.Enabled(context.Background(), slog.LevelWarn) {
		t.Error("WARN should be enabled")
	}

	// INFO is NOT enabled (below handler level)
	if multi.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("INFO should NOT be enabled when min level is WARN")
	}
}

func TestLogDir(t *testing.T) {
	dir := LogDir("my-project")

	if !strings.HasSuffix(dir, filepath.Join(".ssu", "my-project", "logs")) {
		t.Errorf("unexpected LogDir result: %s", dir)
	}
}
