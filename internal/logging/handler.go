// Package logging provides structured file logging with a bash-compatible format.
//
// BracketHandler implements slog.Handler and produces output in the format:
//
//	[2024-01-15 10:30:00] [INFO] message
//
// This matches the log format from the original bash SSU script.
package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
)

// BracketHandler is a slog.Handler that writes log records in bracket format.
// It is safe for concurrent use.
type BracketHandler struct {
	w     io.Writer
	level slog.Leveler
	mu    sync.Mutex
}

// NewBracketHandler creates a BracketHandler that writes to w at the given level.
func NewBracketHandler(w io.Writer, level slog.Level) *BracketHandler {
	return &BracketHandler{w: w, level: level}
}

// Enabled reports whether the handler handles records at the given level.
func (h *BracketHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

// Handle formats and writes a log record in bracket format.
func (h *BracketHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	ts := r.Time.Format("2006-01-02 15:04:05")
	lvl := strings.ToUpper(r.Level.String())
	_, err := fmt.Fprintf(h.w, "[%s] [%s] %s\n", ts, lvl, r.Message)
	return err
}

// WithAttrs returns the handler unchanged. SSU uses simple messages without
// structured attributes in file logs.
func (h *BracketHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return h
}

// WithGroup returns the handler unchanged.
func (h *BracketHandler) WithGroup(_ string) slog.Handler {
	return h
}
