package git

import "testing"

func TestParseSemVer(t *testing.T) {
	tests := []struct {
		input string
		want  *SemVer
	}{
		{"v1.2.3", &SemVer{1, 2, 3, ""}},
		{"v0.0.1", &SemVer{0, 0, 1, ""}},
		{"v10.20.30", &SemVer{10, 20, 30, ""}},
		{"1.2.3", &SemVer{1, 2, 3, ""}},
		{"v1.2.3-beta.1", &SemVer{1, 2, 3, "beta.1"}},
		{"v1.0.0-rc.1", &SemVer{1, 0, 0, "rc.1"}},
		// Invalid
		{"", nil},
		{"v", nil},
		{"v1", nil},
		{"v1.2", nil},
		{"v1.2.3.4", nil},
		{"vfoo.bar.baz", nil},
		{"not-a-version", nil},
	}

	for _, tt := range tests {
		got := ParseSemVer(tt.input)
		if tt.want == nil {
			if got != nil {
				t.Errorf("ParseSemVer(%q) = %v, want nil", tt.input, got)
			}
			continue
		}
		if got == nil {
			t.Errorf("ParseSemVer(%q) = nil, want %v", tt.input, tt.want)
			continue
		}
		if *got != *tt.want {
			t.Errorf("ParseSemVer(%q) = %v, want %v", tt.input, *got, *tt.want)
		}
	}
}

func TestSemVerString(t *testing.T) {
	tests := []struct {
		sv   SemVer
		want string
	}{
		{SemVer{1, 2, 3, ""}, "v1.2.3"},
		{SemVer{0, 0, 1, ""}, "v0.0.1"},
		{SemVer{1, 0, 0, "beta.1"}, "v1.0.0-beta.1"},
	}

	for _, tt := range tests {
		got := tt.sv.String()
		if got != tt.want {
			t.Errorf("SemVer%v.String() = %q, want %q", tt.sv, got, tt.want)
		}
	}
}

func TestIncrementPatch(t *testing.T) {
	tests := []struct {
		input SemVer
		want  SemVer
	}{
		{SemVer{1, 2, 3, ""}, SemVer{1, 2, 4, ""}},
		{SemVer{0, 0, 0, ""}, SemVer{0, 0, 1, ""}},
		{SemVer{1, 0, 0, "beta.1"}, SemVer{1, 0, 1, ""}}, // prerelease cleared
	}

	for _, tt := range tests {
		got := tt.input.IncrementPatch()
		if got != tt.want {
			t.Errorf("SemVer%v.IncrementPatch() = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestSemVerLess(t *testing.T) {
	tests := []struct {
		a, b SemVer
		want bool
	}{
		{SemVer{1, 0, 0, ""}, SemVer{2, 0, 0, ""}, true},
		{SemVer{1, 0, 0, ""}, SemVer{1, 1, 0, ""}, true},
		{SemVer{1, 0, 0, ""}, SemVer{1, 0, 1, ""}, true},
		{SemVer{2, 0, 0, ""}, SemVer{1, 0, 0, ""}, false},
		{SemVer{1, 0, 0, ""}, SemVer{1, 0, 0, ""}, false},
		// Prerelease is less than release
		{SemVer{1, 0, 0, "beta"}, SemVer{1, 0, 0, ""}, true},
		{SemVer{1, 0, 0, ""}, SemVer{1, 0, 0, "beta"}, false},
	}

	for _, tt := range tests {
		got := tt.a.Less(tt.b)
		if got != tt.want {
			t.Errorf("%v.Less(%v) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestLatestSemVer(t *testing.T) {
	tests := []struct {
		tags []string
		want *SemVer
	}{
		{[]string{"v1.0.0", "v1.1.0", "v1.0.5"}, &SemVer{1, 1, 0, ""}},
		{[]string{"v0.0.1"}, &SemVer{0, 0, 1, ""}},
		{[]string{"not-a-tag", "also-not"}, nil},
		{nil, nil},
		{[]string{}, nil},
		{[]string{"v1.0.0", "v1.0.0-beta.1"}, &SemVer{1, 0, 0, ""}}, // release > prerelease of same version
		{[]string{"v2.0.0", "v1.0.0", "v3.0.0"}, &SemVer{3, 0, 0, ""}},
	}

	for _, tt := range tests {
		got := LatestSemVer(tt.tags)
		if tt.want == nil {
			if got != nil {
				t.Errorf("LatestSemVer(%v) = %v, want nil", tt.tags, got)
			}
			continue
		}
		if got == nil {
			t.Errorf("LatestSemVer(%v) = nil, want %v", tt.tags, tt.want)
			continue
		}
		if *got != *tt.want {
			t.Errorf("LatestSemVer(%v) = %v, want %v", tt.tags, *got, *tt.want)
		}
	}
}
