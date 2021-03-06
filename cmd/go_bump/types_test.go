package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnresolvedPackageID_Resolve(t *testing.T) {
	defaultConfig := newDefaultConfig()
	defaultConfig.DefaultProjectShortcut = "github.com/dfuse-io"

	tests := []struct {
		name        string
		in          string
		config      *Config
		expected    string
		expectedErr error
	}{
		{"no replacement", "github.com/dfuse-io/test@develop", defaultConfig, "github.com/dfuse-io/test@develop", nil},
		{"repo replacement", "@dfuse-io/test@develop", defaultConfig, "github.com/dfuse-io/test@develop", nil},
		{"project replacement", "~test@develop", defaultConfig, "github.com/dfuse-io/test@develop", nil},
		{"branch replacement", "github.com/dfuse-io/test!", defaultConfig, "github.com/dfuse-io/test@develop", nil},
		{"branch replacement & repo", "@dfuse-io/test!", defaultConfig, "github.com/dfuse-io/test@develop", nil},
		{"branch replacement & project", "~test!", defaultConfig, "github.com/dfuse-io/test@develop", nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := UnresolvedPackageID(test.in).Resolve(test.config)
			if test.expectedErr == nil {
				require.NoError(t, err)
				assert.Equal(t, PackageID(test.expected), actual)
			} else {
				assert.Equal(t, test.expectedErr, err)
			}
		})
	}
}
