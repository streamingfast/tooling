package main

import (
	"testing"

	require "github.com/stretchr/testify/require"
)

func Test_checkGoModForLocalReplacement(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name                      string
		args                      args
		wantFoundLocalReplacement bool
	}{
		{"go.mod with replacement", args{"./testdata/with_replacement.go.mod"}, true},
		{"go.mod with replacement and toolchain", args{"./testdata/with_replacement_and_toolchain.go.mod"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found := checkGoModForLocalReplacement(tt.args.path)

			if tt.wantFoundLocalReplacement {
				require.True(t, found, "Should have found local replacement in go.mod file")
			} else {
				require.False(t, found, "Should not have found local replacement in go.mod file")
			}
		})
	}
}
