package stack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testSHA = "0123456789abcdef0123456789abcdef01234567"

func TestFileWriteReadRoundTripSorted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stack.json")
	f := New([]Module{
		{Path: "zeta", Branch: "develop", SHA: testSHA},
		{Path: "alpha", Branch: "feature/x", SHA: strings.Repeat("a", 40)},
	})
	if err := Write(path, f); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.Modules[0].Path != "alpha" || got.Modules[1].Path != "zeta" {
		t.Fatalf("modules not sorted: %#v", got.Modules)
	}
	data, _ := os.ReadFile(path)
	if !strings.HasSuffix(string(data), "\n") {
		t.Error("stack file should end with newline")
	}
}

func TestFileValidateRejectsUnsafeAndDuplicateEntries(t *testing.T) {
	tests := []struct {
		name    string
		modules []Module
	}{
		{"traversal", []Module{{Path: "../outside", SHA: testSHA}}},
		{"absolute", []Module{{Path: "/outside", SHA: testSHA}}},
		{"bad sha", []Module{{Path: "module", SHA: "abc"}}},
		{"duplicate", []Module{{Path: "module", SHA: testSHA}, {Path: "module", SHA: testSHA}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := (&File{Version: Version, Modules: tt.modules}).Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestReadRejectsUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stack.json")
	data := `{"version":1,"generated_at":"now","extra":true,"modules":[{"path":"module","sha":"` + testSHA + `"}]}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(path); err == nil {
		t.Fatal("expected unknown field error")
	}
}
