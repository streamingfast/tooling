package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	. "github.com/streamingfast/cli"
	"github.com/streamingfast/tooling/cli"
)

func main() {
	Run(
		"rate_of [-i <interval>] <timestamp> ...",
		"Compute the rate at which input has arrived based on each timestamp of it",
		Description(`
			This command can be used to compute the rate of something that happened by extracting
			the rate from succesive timestamps (actual date part elided for brievity).

			  00:52:02
			  00:52:03
			  00:52:04
			  00:53:04
			  00:54:35

			At an interval of 1m (default) would give the following rates:

			  1 msg/s
			  1 msg/s

			The command is able to parse a varities of timestamp input, multiple messages per interval is
			supported as well as when there is less than one message per interval.
		`),
		ArbitraryArgs(),
		Flags(func(flags *pflag.FlagSet) {
			flags.DurationP("interval", "i", time.Minute, "Interval at which we want to compute rate for")
		}),
		Example(`
			# Rate per minute for those 3 timestamps
			rate_of 2022-10-21T00:56:28.482759866-04:00 2022-10-21T00:56:42.54522114-04:00 2022-10-21T00:56:56.418039554-04:00

			# Rate per second for those 3 timestamps
			rate_of -i 1m 2022-10-21T00:56:28.482759866-04:00 2022-10-21T00:56:42.54522114-04:00 2022-10-21T00:56:56.418039554-04:00

			# Rate per hour from a file processed through jq
			cat <file> | jq .timestamp | rate_of -i 1h
		`),
		Execute(func(cmd *cobra.Command, args []string) error {
			interval, _ := cmd.Flags().GetDuration("interval")

			return execute(interval, args, func(line string) { fmt.Println(line) })
		}),
	)
}

func execute(interval time.Duration, args []string, out func(line string)) error {
	intervalUnit := intervalUnitString(interval)

	var activeBucket *int64
	var activeCount uint64
	var lastTimestamp *time.Time
	var totalCount uint64

	scanner := cli.NewArgumentScanner(args)
	for element, ok := scanner.ScanArgument(); ok; element, ok = scanner.ScanArgument() {
		timestamp := toTimestamp(element)
		nanos := timestamp.UnixNano()
		bucket := nanos - (nanos % int64(interval))

		// We reached a new bucket or it's our first ever element
		if activeBucket == nil || bucket != *activeBucket {
			if activeBucket != nil {
				bucketDirecltyFollowsActiveBucket := bucket-*activeBucket == int64(interval)

				if !bucketDirecltyFollowsActiveBucket {
					// `lastTimestamp` is always non-nil here because `activeBucket` is set which means we processed already at least one message
					duration := timestamp.Sub(*lastTimestamp)

					out(fmt.Sprintf("%.3g msg/%s", float64(interval)/float64(duration), intervalUnit))
				} else {
					out(fmt.Sprintf("%d msg/%s", activeCount, intervalUnit))
				}
			}

			activeBucket = &bucket
			activeCount = 0
		}

		activeCount++
		totalCount++

		lastTimestamp = &timestamp
	}

	if activeCount > 1 {
		out(fmt.Sprintf("%d msg/%s", activeCount, intervalUnit))
	}

	cli.Ensure(totalCount != 1, "You only provided one timestamp, it's not possible to infer rate from a single timestamp value")
	return nil
}

func toTimestamp(element string) time.Time {
	parsed, _, ok := cli.ParseDateLikeInput(element, cli.DateLikeHintNone)
	cli.Ensure(ok, "Unable to interpret %q as a timestamp input", element)

	return parsed
}

func intervalUnitString(interval time.Duration) string {
	switch interval {
	case 1 * time.Millisecond:
		return "ms"
	case 1 * time.Second:
		return "s"
	case 1 * time.Minute:
		return "min"
	case 1 * time.Hour:
		return "hour"
	case 24 * time.Hour:
		return "day"
	default:
		return interval.String()
	}
}
