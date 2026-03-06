package models

import (
	"testing"
)

func TestValidateVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		wantErr bool
	}{
		// Valid: major only
		{"major only", "1", false},
		{"major zero", "0", false},
		{"major large", "123", false},

		// Valid: major.minor
		{"major.minor", "1.2", false},
		{"major.minor zeros", "0.0", false},

		// Valid: full semver
		{"full semver", "1.2.3", false},
		{"full semver zeros", "0.0.0", false},
		{"full semver large", "10.20.30", false},

		// Valid: pre-release on all forms
		{"pre-release major only", "1-alpha", false},
		{"pre-release major.minor", "1.2-beta", false},
		{"pre-release full semver", "1.2.3-rc.1", false},
		{"pre-release numeric", "1.0.0-0.3.7", false},
		{"pre-release alphanumeric", "1.0.0-x.7.z.92", false},

		// Valid: build metadata on all forms
		{"build metadata major only", "1+build.1", false},
		{"build metadata major.minor", "1.2+build.1", false},
		{"build metadata full semver", "1.2.3+build.123", false},

		// Valid: pre-release + build metadata
		{"pre-release and build", "1.2-beta+build.42", false},
		{"full pre-release and build", "1.2.3-alpha+001", false},

		// Invalid: empty
		{"empty string", "", true},

		// Invalid: non-numeric
		{"non-numeric", "a.b.c", true},
		{"alpha only", "abc", true},

		// Invalid: too many segments
		{"four segments", "1.2.3.4", true},

		// Invalid: negative / leading minus
		{"negative", "-1", true},

		// Invalid: leading zeros
		{"leading zero major", "01.2.3", true},
		{"leading zero minor", "1.02.3", true},
		{"leading zero patch", "1.2.03", true},

		// Invalid: v prefix
		{"v prefix", "v1.0.0", true},

		// Invalid: trailing dot
		{"trailing dot", "1.", true},
		{"trailing dots", "1.2.", true},

		// Invalid: whitespace
		{"leading space", " 1.0.0", true},
		{"trailing space", "1.0.0 ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVersion(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateVersion(%q) error = %v, wantErr %v", tt.version, err, tt.wantErr)
			}
		})
	}
}
