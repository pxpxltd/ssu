package compat

import (
	"bytes"
	"strings"
	"testing"
)

func TestCheckOldFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		detected bool
		contains string // expected substring in output (only checked when detected)
	}{
		{
			name:     "--status is detected",
			args:     []string{"ssu", "--status"},
			detected: true,
			contains: "ssu status",
		},
		{
			name:     "--push is detected",
			args:     []string{"ssu", "--push"},
			detected: true,
			contains: "ssu push",
		},
		{
			name:     "--rollback is detected",
			args:     []string{"ssu", "--rollback"},
			detected: true,
			contains: "ssu rollback",
		},
		{
			name:     "--auto is detected",
			args:     []string{"ssu", "--auto"},
			detected: true,
			contains: "ssu update --auto",
		},
		{
			name:     "--dry-run is detected",
			args:     []string{"ssu", "--dry-run"},
			detected: true,
			contains: "ssu update --dry-run",
		},
		{
			name:     "valid subcommand is not detected",
			args:     []string{"ssu", "status"},
			detected: false,
		},
		{
			name:     "no args is not detected",
			args:     []string{"ssu"},
			detected: false,
		},
		{
			name:     "old flag as subcommand flag is not detected",
			args:     []string{"ssu", "update", "--dry-run"},
			detected: false,
		},
		{
			name:     "unknown flag is not detected",
			args:     []string{"ssu", "--verbose"},
			detected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			got := CheckOldFlags(tt.args, &buf)

			if got != tt.detected {
				t.Errorf("CheckOldFlags(%v) = %v, want %v", tt.args, got, tt.detected)
			}

			if tt.detected && !strings.Contains(buf.String(), tt.contains) {
				t.Errorf("output %q does not contain %q", buf.String(), tt.contains)
			}

			if tt.detected && !strings.Contains(buf.String(), "ssu help") {
				t.Errorf("output %q does not contain help hint", buf.String())
			}

			if !tt.detected && buf.Len() > 0 {
				t.Errorf("expected no output when not detected, got %q", buf.String())
			}
		})
	}
}
