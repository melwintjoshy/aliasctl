package versions

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		output   string
		expected Version
	}{
		{"go version go1.27.1 darwin/arm64", Version{1, 27, 1}},
		{"Client Version: v1.36.4\nKustomize Version: v5.8.1", Version{1, 36, 4}},
		{"v4.2.4+g3900f43", Version{4, 2, 4}},
		{"v20.20.2", Version{20, 20, 2}},
		{"Python 3.14.6", Version{3, 14, 6}},
		{"Docker version 29.8.0, build 88096ef", Version{29, 8, 0}},
		{"git version 2.55.0", Version{2, 55, 0}},
		{"jq-1.8.2", Version{1, 8, 2}},
		{"python3 3.12.1", Version{3, 12, 1}},
		{"tool 7", Version{7}},
	}

	for _, tt := range tests {
		got, ok := Parse(tt.output)

		if !ok || !reflect.DeepEqual(got, tt.expected) {
			t.Fatalf("Parse(%q) = %v, %v; want %v", tt.output, got, ok, tt.expected)
		}
	}

	if _, ok := Parse("Unable to locate a Java Runtime."); ok {
		t.Fatal("expected no version in text without digits")
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		a, b     Version
		expected int
	}{
		{Version{1, 22}, Version{1, 22, 0}, 0},
		{Version{1, 22, 1}, Version{1, 22}, 1},
		{Version{1, 9}, Version{1, 10}, -1},
		{Version{2}, Version{1, 99, 99}, 1},
	}

	for _, tt := range tests {
		if got := Compare(tt.a, tt.b); got != tt.expected {
			t.Fatalf("Compare(%v, %v) = %d; want %d", tt.a, tt.b, got, tt.expected)
		}
	}
}

func TestConstraintMatch(t *testing.T) {
	tests := []struct {
		rule    string
		version Version
		match   bool
	}{
		{"*", Version{0, 1}, true},
		{"1.22", Version{1, 22, 4}, true},
		{"1.22", Version{1, 22}, true},
		{"1.22", Version{1, 23, 0}, false},
		{"1.22", Version{1, 2}, false},
		{"1.22.4", Version{1, 22, 4}, true},
		{"1.22.4", Version{1, 22, 5}, false},
		{"=1.22", Version{1, 22, 0}, true},
		{"=1.22", Version{1, 22, 4}, false},
		{"v1.22", Version{1, 22, 4}, true},
		{">=v1.29 <v1.31", Version{1, 30, 2}, true},
		{">=1.6", Version{1, 6}, true},
		{">=1.6", Version{1, 5, 9}, false},
		{">1.6", Version{1, 6, 0}, false},
		{"<=2", Version{2, 0, 0}, true},
		{"<2", Version{1, 99}, true},
		{">=1.29 <1.31", Version{1, 30, 2}, true},
		{">=1.29 <1.31", Version{1, 31, 0}, false},
		{">=1.29 <1.31", Version{1, 28, 9}, false},
	}

	for _, tt := range tests {
		constraint, err := ParseConstraint(tt.rule)
		if err != nil {
			t.Fatalf("ParseConstraint(%q): %v", tt.rule, err)
		}

		if got := constraint.Match(tt.version); got != tt.match {
			t.Fatalf("%q.Match(%v) = %v; want %v", tt.rule, tt.version, got, tt.match)
		}
	}
}

func TestParseConstraintErrors(t *testing.T) {
	for _, rule := range []string{"", "  ", "~>1.2", "latest", ">=", "1..2", "1.x", "* >1", "v", ">=v"} {
		if _, err := ParseConstraint(rule); err == nil {
			t.Fatalf("expected error for %q", rule)
		}
	}

	// a space after the operator is the common slip, so the message says what to do
	_, err := ParseConstraint(">= 1.29")

	expected := `operator ">=" must be followed by a version, e.g. >=1.29`

	if err == nil || err.Error() != expected {
		t.Fatalf("expected %q, got %v", expected, err)
	}
}
