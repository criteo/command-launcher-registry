package principal

import (
	"errors"
	"slices"
	"testing"
)

func TestParseList(t *testing.T) {
	tests := []struct {
		name  string
		texts []string
		want  []string
	}{
		{"comma and space", []string{"alice, team-build ops"}, []string{"alice", "team-build", "ops"}},
		{"empty entries dropped", []string{" a,,b "}, []string{"a", "b"}},
		{"tabs and newlines", []string{"a\tb\nc\r\nd"}, []string{"a", "b", "c", "d"}},
		{"empty input", []string{""}, nil},
		{"only separators", []string{" , ,\t"}, nil},
		{"no input", nil, nil},
		{"several texts keep order", []string{"alice, team-build", "ops"}, []string{"alice", "team-build", "ops"}},
		{"duplicates kept", []string{"a a"}, []string{"a", "a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseList(tt.texts...)
			if !slices.Equal(got, tt.want) {
				t.Errorf("ParseList(%q) = %q, want %q", tt.texts, got, tt.want)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		in   string
		want error
	}{
		{"alice", nil},
		{"team-build", nil},
		{"a.mazurov@criteo.com", nil},
		{"", ErrEmpty},
		{"a,b", ErrHasComma},
		{"a b", ErrHasWhitespace},
		{" alice", ErrHasWhitespace},
		{"a\tb", ErrHasWhitespace},
		{"a\nb", ErrHasWhitespace},
		{"a b", ErrHasWhitespace},
	}
	for _, tt := range tests {
		if got := Validate(tt.in); !errors.Is(got, tt.want) {
			t.Errorf("Validate(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

// Every principal ParseList returns passes Validate.
func TestParseListOutputIsValid(t *testing.T) {
	for _, p := range ParseList("alice, team-build ops\t\n,x y") {
		if err := Validate(p); err != nil {
			t.Errorf("ParseList returned invalid principal %q: %v", p, err)
		}
	}
}

func TestMatches(t *testing.T) {
	list := []string{"alice", "team-build"}
	tests := []struct {
		name     string
		list     []string
		username string
		groups   []string
		want     bool
	}{
		{"username", list, "alice", nil, true},
		{"group", list, "bob", []string{"ops", "team-build"}, true},
		{"no match", list, "bob", []string{"ops"}, false},
		{"case-sensitive", list, "Alice", []string{"Team-Build"}, false},
		{"group name is not a username", list, "team-build-x", []string{"team"}, false},
		{"empty list", nil, "alice", []string{"team-build"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Matches(tt.list, tt.username, tt.groups); got != tt.want {
				t.Errorf("Matches(%q, %q, %q) = %v, want %v", tt.list, tt.username, tt.groups, got, tt.want)
			}
		})
	}
}
