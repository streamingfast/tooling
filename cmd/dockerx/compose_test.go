package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseComposeProjects(t *testing.T) {
	tests := []struct {
		name        string
		in          string
		expected    []composeProject
		expectedErr string
	}{
		{
			"empty array",
			`[]`,
			nil,
			"",
		},
		{
			"empty output",
			"\n",
			nil,
			"",
		},
		{
			"single project",
			`[{"Name":"api","Status":"running(2)","ConfigFiles":"/w/api/docker-compose.yml"}]`,
			[]composeProject{{Name: "api", Status: "running(2)", ConfigFiles: "/w/api/docker-compose.yml"}},
			"",
		},
		{
			"multiple projects are sorted by name",
			`[{"Name":"zeta","Status":"running(1)"},{"Name":"alpha","Status":"running(3)"}]`,
			[]composeProject{{Name: "alpha", Status: "running(3)"}, {Name: "zeta", Status: "running(1)"}},
			"",
		},
		{
			"projects without a name are ignored",
			`[{"Name":"","Status":"running(1)"},{"Name":"api","Status":"running(1)"}]`,
			[]composeProject{{Name: "api", Status: "running(1)"}},
			"",
		},
		{
			"invalid json",
			`not json`,
			nil,
			"decoding 'docker compose ls' output: invalid character 'o' in literal null (expecting 'u')",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := parseComposeProjects([]byte(test.in))
			if test.expectedErr == "" {
				require.NoError(t, err)
				assert.Equal(t, test.expected, actual)
			} else {
				require.Error(t, err)
				assert.Equal(t, test.expectedErr, err.Error())
			}
		})
	}
}

func TestPartitionProjects(t *testing.T) {
	projects := []composeProject{
		{Name: "alpha"},
		{Name: "beta"},
		{Name: "gamma"},
	}

	tests := []struct {
		name            string
		except          []string
		expectedKept    []string
		expectedSkipped []string
	}{
		{"no exception", nil, []string{"alpha", "beta", "gamma"}, nil},
		{"one exception", []string{"beta"}, []string{"alpha", "gamma"}, []string{"beta"}},
		{"many exceptions", []string{"beta", "alpha"}, []string{"gamma"}, []string{"alpha", "beta"}},
		{"exception is trimmed", []string{"  beta  "}, []string{"alpha", "gamma"}, []string{"beta"}},
		{"empty exception is ignored", []string{""}, []string{"alpha", "beta", "gamma"}, nil},
		{"unknown exception matches nothing", []string{"delta"}, []string{"alpha", "beta", "gamma"}, nil},
		{"exception is case sensitive", []string{"BETA"}, []string{"alpha", "beta", "gamma"}, nil},
		{"all excepted", []string{"alpha", "beta", "gamma"}, nil, []string{"alpha", "beta", "gamma"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			kept, skipped := partitionProjects(projects, test.except)

			assert.Equal(t, test.expectedKept, projectNames(kept))
			assert.Equal(t, test.expectedSkipped, projectNames(skipped))
		})
	}
}

func TestComposeDownArgs(t *testing.T) {
	tests := []struct {
		name        string
		project     string
		keepVolumes bool
		expected    []string
	}{
		{"removes volumes by default", "api", false, []string{"compose", "--project-name", "api", "down", "--volumes"}},
		{"keeps volumes", "api", true, []string{"compose", "--project-name", "api", "down"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, composeDownArgs(test.project, test.keepVolumes))
		})
	}
}

func projectNames(in []composeProject) (out []string) {
	for _, project := range in {
		out = append(out, project.Name)
	}
	return out
}
