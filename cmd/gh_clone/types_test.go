package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnresolvedPackageID_Resolve(t *testing.T) {
	defaultConfig := newDefaultConfig()

	tests := []struct {
		name        string
		in          string
		config      *Config
		expected    string
		expectedErr error
	}{
		{"org/repo", "acme/example", defaultConfig, "https://github.com/acme/example.git", nil},

		{"github.com/org/repo", "github.com/acme/example", defaultConfig, "https://github.com/acme/example.git", nil},
		{"github.com/org/repo.git", "github.com/acme/example.git", defaultConfig, "https://github.com/acme/example.git", nil},

		{"https://github.com/org/repo", "https://github.com/acme/example", defaultConfig, "https://github.com/acme/example.git", nil},
		{"https://github.com/org/repo.git", "https://github.com/acme/example.git", defaultConfig, "https://github.com/acme/example.git", nil},

		{"git@github.com:org/repo", "git@github.com:acme/example", defaultConfig, "git@github.com:acme/example.git", nil},
		{"git@github.com:org/repo.git", "git@github.com:acme/example.git", defaultConfig, "git@github.com:acme/example.git", nil},

		{"invalid input", "", defaultConfig, "", fmt.Errorf("invalid input %q: not long enough", "")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := UnresolvedInput(test.in).Resolve(test.config)
			if test.expectedErr == nil {
				require.NoError(t, err)
				assert.Equal(t, ResolvedInput{Organization: "acme", Repository: "example", Expanded: test.expected}, actual, "Wrong input %q", test.in)
			} else {
				assert.Equal(t, test.expectedErr, err, "Wrong input %q", test.in)
			}
		})
	}
}
