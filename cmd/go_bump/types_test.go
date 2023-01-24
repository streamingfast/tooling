package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnresolvedPackageID_Resolve(t *testing.T) {
	defaultConfig := newDefaultConfig()
	defaultConfig.DefaultBranchShortcut = "@custom"
	defaultConfig.DefaultProjectShortcut = "github.com/streamingfast"

	tests := []struct {
		name        string
		in          string
		config      *Config
		expected    string
		expectedErr error
	}{
		{"no replacement", "github.com/streamingfast/test@develop", defaultConfig, "github.com/streamingfast/test@develop", nil},
		{"repo replacement", "@streamingfast/test@develop", defaultConfig, "github.com/streamingfast/test@develop", nil},
		{"project replacement", "~test@develop", defaultConfig, "github.com/streamingfast/test@develop", nil},
		{"branch replacement", "github.com/streamingfast/test!", defaultConfig, "github.com/streamingfast/test@custom", nil},
		{"branch replacement & repo", "@streamingfast/test!", defaultConfig, "github.com/streamingfast/test@custom", nil},
		{"branch replacement & project", "~test!", defaultConfig, "github.com/streamingfast/test@custom", nil},
		{"plain dep", "test", defaultConfig, "github.com/streamingfast/test@custom", nil},
		{"plain dep with manual branch", "test@develop", defaultConfig, "github.com/streamingfast/test@develop", nil},
		{"plain dep with version", "test@v0.1.0", defaultConfig, "github.com/streamingfast/test@v0.1.0", nil},

		// FIXME: This should work somehow, might be hard to make the interpretation right ....
		// {"plain dep with manual  namespaced branch", "project@namespace/develop", defaultConfig, "github.com/project/test@namespace/develop", nil},
		{"monorepo project + name dep", "project/test", defaultConfig, "github.com/streamingfast/project/test@custom", nil},
		{"monorepo project + name dep with manual branch", "project/test@develop", defaultConfig, "github.com/streamingfast/project/test@develop", nil},

		{"project + name dep", "@project/test", defaultConfig, "github.com/project/test@custom", nil},
		{"project + name dep with manual branch", "@project/test@develop", defaultConfig, "github.com/project/test@develop", nil},
		// FIXME: This should work somehow, might be hard to make the interpretation right ....
		// {"project + name dep with manual namespaced branch", "project/test@namespace/develop", defaultConfig, "github.com/project/test@namespace/develop", nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := UnresolvedPackageID(test.in).Resolve(test.config)
			if test.expectedErr == nil {
				require.NoError(t, err)
				assert.Equal(t, PackageID(test.expected), actual, "Wrong input %q", test.in)
			} else {
				assert.Equal(t, test.expectedErr, err, "Wrong input %q", test.in)
			}
		})
	}
}
