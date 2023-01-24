package main

import (
	"context"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func Test_execute(t *testing.T) {
	tests := []struct {
		name      string
		interval  time.Duration
		args      []string
		want      []string
		assertion require.ErrorAssertionFunc
	}{
		{
			"empty args",
			time.Second,
			nil,
			nil,
			require.NoError,
		},
		// We use cli.Ensure which os.Exit(1) and this breaks the test, could we tweak cli to have a test convenience that would panic instead?
		// {
		// 	"single element error out",
		// 	[]string{
		// 		"2022-01-01T00:00:00.000000-04:00",
		// 	},
		// 	nil,
		// 	require.NoError,
		// },
		{
			"multiple by 1s rate",
			time.Second,
			[]string{
				"2022-01-01T00:00:00.000000-04:00",
				"2022-01-01T00:00:00.250000-04:00",
				"2022-01-01T00:00:00.500000-04:00",
				"2022-01-01T00:00:00.750000-04:00",
				"2022-01-01T00:00:01.000000-04:00",
				"2022-01-01T00:00:01.330000-04:00",
				"2022-01-01T00:00:02.4500000-04:00",
				"2022-01-01T00:00:03.000000-04:00",
				"2022-01-01T00:00:03.120000-04:00",
			},
			[]string{
				"4 msg/s",
				"2 msg/s",
				"1 msg/s",
				"2 msg/s",
			},
			require.NoError,
		},
		{
			"perfect 1s rate",
			time.Second,
			[]string{
				"2022-01-01T00:00:00.000000-04:00",
				"2022-01-01T00:00:01.000000-04:00",
				"2022-01-01T00:00:02.000000-04:00",
				"2022-01-01T00:00:03.000000-04:00",
			},
			[]string{
				"1 msg/s",
				"1 msg/s",
				"1 msg/s",
			},
			require.NoError,
		},
		{
			"over 1s rate",
			time.Second,
			[]string{
				"2022-01-01T00:00:00.000000-04:00",
				"2022-01-01T00:00:02.000000-04:00",
				"2022-01-01T00:00:04.000000-04:00",
				"2022-01-01T00:00:06.000000-04:00",
			},
			[]string{
				"0.5 msg/s",
				"0.5 msg/s",
				"0.5 msg/s",
			},
			require.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.ExecuteContext(context.Background())

			var outputLines []string
			err := execute(tt.interval, tt.args, func(line string) { outputLines = append(outputLines, line) })

			tt.assertion(t, err)
			require.Equal(t, tt.want, outputLines)
		})
	}
}
