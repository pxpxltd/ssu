package stack

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/pxpxltd/ssu/internal/backup"
)

const Version = 1

var shaPattern = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)

type Module struct {
	Path   string `json:"path"`
	Branch string `json:"branch,omitempty"`
	SHA    string `json:"sha"`
}

type File struct {
	Version     int      `json:"version"`
	GeneratedAt string   `json:"generated_at"`
	Modules     []Module `json:"modules"`
}

func New(modules []Module) *File {
	sort.Slice(modules, func(i, j int) bool { return modules[i].Path < modules[j].Path })
	return &File{
		Version:     Version,
		GeneratedAt: time.Now().Format(time.RFC3339),
		Modules:     modules,
	}
}

func Read(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading stack file: %w", err)
	}
	var f File
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return nil, fmt.Errorf("parsing stack file: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err == nil {
		return nil, fmt.Errorf("parsing stack file: multiple JSON values")
	} else if err != io.EOF {
		return nil, fmt.Errorf("parsing stack file: trailing data: %w", err)
	}
	if err := f.Validate(); err != nil {
		return nil, err
	}
	return &f, nil
}

func (f *File) Validate() error {
	if f.Version != Version {
		return fmt.Errorf("unsupported stack file version %d (supported: %d)", f.Version, Version)
	}
	if len(f.Modules) == 0 {
		return fmt.Errorf("stack file contains no modules")
	}
	seen := make(map[string]bool, len(f.Modules))
	for _, m := range f.Modules {
		clean := filepath.Clean(m.Path)
		if m.Path == "" || filepath.IsAbs(m.Path) || clean != m.Path || clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return fmt.Errorf("invalid module path %q", m.Path)
		}
		if seen[m.Path] {
			return fmt.Errorf("duplicate module path %q", m.Path)
		}
		seen[m.Path] = true
		if !shaPattern.MatchString(m.SHA) {
			return fmt.Errorf("invalid SHA for %s: expected 40 hexadecimal characters", m.Path)
		}
		if m.Branch != "" && invalidBranch(m.Branch) {
			return fmt.Errorf("invalid branch for %s: %q", m.Path, m.Branch)
		}
	}
	return nil
}

func invalidBranch(branch string) bool {
	return strings.HasPrefix(branch, "-") ||
		strings.HasSuffix(branch, "/") ||
		strings.HasSuffix(branch, ".") ||
		strings.Contains(branch, "..") ||
		strings.Contains(branch, "@{") ||
		strings.Contains(branch, "//") ||
		strings.ContainsAny(branch, " ~^:?*[\\\t\r\n")
}

func Write(path string, f *File) error {
	if err := f.Validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding stack file: %w", err)
	}
	data = append(data, '\n')
	if err := backup.AtomicWrite(path, data, 0644); err != nil {
		return fmt.Errorf("writing stack file: %w", err)
	}
	return nil
}
