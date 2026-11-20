// Package principal holds the text format of role principals (ADR 0001).
//
// A principal is a username or a group name. Wherever principals are given
// as text (server setting, CLI flags), the value is a list separated by
// commas, whitespace, or both, and empty entries are dropped. The API and
// storage keep one principal per JSON array element, so a principal never
// contains a comma or whitespace.
//
// Both the server and the CLI import this package, so the rule cannot drift
// between them.
package principal

import (
	"errors"
	"strings"
	"unicode"
)

// Errors returned by Validate.
var (
	ErrEmpty         = errors.New("must not be empty")
	ErrHasComma      = errors.New("must not contain a comma")
	ErrHasWhitespace = errors.New("must not contain whitespace")
)

// ParseList splits each text on commas and whitespace and returns the
// principals in order, dropping empty entries. It returns nil when no
// principal is found.
func ParseList(texts ...string) []string {
	var out []string
	for _, text := range texts {
		out = append(out, strings.FieldsFunc(text, isSeparator)...)
	}
	return out
}

// Validate checks one principal as stored in an API array element: it must
// be non-empty and contain no comma and no whitespace.
func Validate(p string) error {
	if p == "" {
		return ErrEmpty
	}
	if strings.ContainsRune(p, ',') {
		return ErrHasComma
	}
	if strings.IndexFunc(p, unicode.IsSpace) >= 0 {
		return ErrHasWhitespace
	}
	return nil
}

func isSeparator(r rune) bool {
	return r == ',' || unicode.IsSpace(r)
}
