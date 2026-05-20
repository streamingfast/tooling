package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseUpgradeEntries(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []upgradeEntry
	}{
		{
			name:     "empty output",
			input:    "",
			expected: nil,
		},
		{
			name:     "no upgrades",
			input:    "go: downloading github.com/foo/bar v1.0.0\n",
			expected: nil,
		},
		{
			name:  "single upgrade",
			input: "go: upgraded github.com/streamingfast/bstream v0.0.2-0.20230101 => v0.0.2-0.20231201\n",
			expected: []upgradeEntry{
				{
					module:      "github.com/streamingfast/bstream",
					fromVersion: "v0.0.2-0.20230101",
					toVersion:   "v0.0.2-0.20231201",
				},
			},
		},
		{
			name: "multiple upgrades",
			input: "go: upgraded github.com/foo/bar v1.0.0 => v1.1.0\n" +
				"go: upgraded github.com/baz/qux v2.3.4 => v2.4.0\n",
			expected: []upgradeEntry{
				{module: "github.com/foo/bar", fromVersion: "v1.0.0", toVersion: "v1.1.0"},
				{module: "github.com/baz/qux", fromVersion: "v2.3.4", toVersion: "v2.4.0"},
			},
		},
		{
			name: "mixed output lines",
			input: "go: downloading github.com/foo/bar v1.1.0\n" +
				"go: upgraded github.com/foo/bar v1.0.0 => v1.1.0\n" +
				"some other line\n",
			expected: []upgradeEntry{
				{module: "github.com/foo/bar", fromVersion: "v1.0.0", toVersion: "v1.1.0"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseUpgradeEntries(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestModuleShortName(t *testing.T) {
	tests := []struct {
		name     string
		module   string
		expected string
	}{
		{
			name:     "simple module",
			module:   "github.com/streamingfast/bstream",
			expected: "bstream",
		},
		{
			name:     "module with version suffix",
			module:   "github.com/streamingfast/bstream@develop",
			expected: "bstream",
		},
		{
			name:     "hyphenated module",
			module:   "github.com/streamingfast/firehose-core",
			expected: "firehose-core",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, moduleShortName(tt.module))
		})
	}
}

func TestPreBumpBranchName(t *testing.T) {
	tests := []struct {
		name       string
		packageIDs []PackageID
		expected   string
	}{
		{
			name:       "single package with version",
			packageIDs: []PackageID{"github.com/streamingfast/bstream@develop"},
			expected:   "bump/bstream",
		},
		{
			name:       "single package without version",
			packageIDs: []PackageID{"github.com/streamingfast/firehose-core"},
			expected:   "bump/firehose-core",
		},
		{
			name: "two packages",
			packageIDs: []PackageID{
				"github.com/streamingfast/bstream@develop",
				"github.com/streamingfast/dstore@develop",
			},
			expected: "bump/bstream-dstore",
		},
		{
			name: "three packages",
			packageIDs: []PackageID{
				"github.com/streamingfast/bstream@develop",
				"github.com/streamingfast/dstore@develop",
				"github.com/streamingfast/firehose-core@develop",
			},
			expected: "bump/bstream-dstore-firehose-core",
		},
		{
			name: "four packages uses generic name",
			packageIDs: []PackageID{
				"github.com/streamingfast/bstream@develop",
				"github.com/streamingfast/dstore@develop",
				"github.com/streamingfast/firehose-core@develop",
				"github.com/streamingfast/substreams@develop",
			},
			expected: "bump/dependencies",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, preBumpBranchName(tt.packageIDs))
		})
	}
}

func TestFinalBranchName(t *testing.T) {
	tests := []struct {
		name       string
		preBranch  string
		upgrades   []upgradeEntry
		packageIDs []PackageID
		expected   string
	}{
		{
			name:      "single package gets version appended",
			preBranch: "bump/dstore",
			upgrades: []upgradeEntry{
				{module: "github.com/streamingfast/dstore", toVersion: "v0.2.4-0.20260520032149-6421410d7faa"},
			},
			packageIDs: []PackageID{"github.com/streamingfast/dstore@develop"},
			expected:   "bump/dstore-to-v0.2.4-0.20260520032149-6421410d7faa",
		},
		{
			name:      "single package no upgrades returns pre-bump name",
			preBranch: "bump/dstore",
			upgrades:  nil,
			packageIDs: []PackageID{"github.com/streamingfast/dstore@develop"},
			expected:  "bump/dstore",
		},
		{
			name:      "multiple packages returns pre-bump name",
			preBranch: "bump/bstream-dstore",
			upgrades: []upgradeEntry{
				{module: "github.com/streamingfast/bstream", toVersion: "v0.1.0"},
				{module: "github.com/streamingfast/dstore", toVersion: "v0.2.0"},
			},
			packageIDs: []PackageID{
				"github.com/streamingfast/bstream@develop",
				"github.com/streamingfast/dstore@develop",
			},
			expected: "bump/bstream-dstore",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := finalBranchName(tt.preBranch, tt.upgrades, tt.packageIDs)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestPRCommitMessage(t *testing.T) {
	tests := []struct {
		name     string
		entries  []upgradeEntry
		expected string
	}{
		{
			name:     "no entries",
			entries:  nil,
			expected: "chore: bump dependencies",
		},
		{
			name: "single entry",
			entries: []upgradeEntry{
				{module: "github.com/foo/bar", fromVersion: "v1.0.0", toVersion: "v1.1.0"},
			},
			expected: "chore: bump github.com/foo/bar v1.0.0 => v1.1.0",
		},
		{
			name: "multiple entries",
			entries: []upgradeEntry{
				{module: "github.com/foo/bar", fromVersion: "v1.0.0", toVersion: "v1.1.0"},
				{module: "github.com/baz/qux", fromVersion: "v2.3.4", toVersion: "v2.4.0"},
			},
			expected: "chore: bump dependencies\n\n- github.com/foo/bar v1.0.0 => v1.1.0\n- github.com/baz/qux v2.3.4 => v2.4.0\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, prCommitMessage(tt.entries))
		})
	}
}

func TestNormalizeGitHubRepo(t *testing.T) {
	tests := []struct {
		name      string
		remoteURL string
		expected  string
	}{
		{
			name:      "https with .git suffix",
			remoteURL: "https://github.com/streamingfast/tooling.git",
			expected:  "streamingfast/tooling",
		},
		{
			name:      "https without .git suffix",
			remoteURL: "https://github.com/streamingfast/tooling",
			expected:  "streamingfast/tooling",
		},
		{
			name:      "ssh with .git suffix",
			remoteURL: "git@github.com:streamingfast/tooling.git",
			expected:  "streamingfast/tooling",
		},
		{
			name:      "ssh without .git suffix",
			remoteURL: "git@github.com:streamingfast/tooling",
			expected:  "streamingfast/tooling",
		},
		{
			name:      "non-github remote",
			remoteURL: "https://gitlab.com/streamingfast/tooling.git",
			expected:  "",
		},
		{
			name:      "empty",
			remoteURL: "",
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizeGitHubRepo(tt.remoteURL))
		})
	}
}

func TestPRURLFromRemote(t *testing.T) {
	tests := []struct {
		name       string
		remoteURL  string
		baseBranch string
		headBranch string
		expected   string
	}{
		{
			name:       "github https remote",
			remoteURL:  "https://github.com/streamingfast/tooling.git",
			baseBranch: "master",
			headBranch: "bump/bstream-to-v0.1.0",
			expected:   "https://github.com/streamingfast/tooling/compare/master...bump/bstream-to-v0.1.0",
		},
		{
			name:       "github ssh remote",
			remoteURL:  "git@github.com:streamingfast/tooling.git",
			baseBranch: "develop",
			headBranch: "bump/bstream-to-v0.1.0",
			expected:   "https://github.com/streamingfast/tooling/compare/develop...bump/bstream-to-v0.1.0",
		},
		{
			name:       "non-github remote returns empty",
			remoteURL:  "https://gitlab.com/foo/bar.git",
			baseBranch: "master",
			headBranch: "bump/bstream",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := prURLFromRemote(tt.remoteURL, tt.baseBranch, tt.headBranch)
			assert.Equal(t, tt.expected, got)
		})
	}
}
