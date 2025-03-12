package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/streamingfast/tooling/cli"
)

var asUnixSecondsFlag = flag.Bool("s", false, "Avoid heuristics based to determine decimal range value and assume it's UNIX seconds since epoch")
var asUnixMillisFlag = flag.Bool("ms", false, "Avoid heuristics based to determine decimal range value and assume it's UNIX milliseconds since epoch")
var timezoneFlag = flag.String("timezone", "local", "When the provided date is not timezone aware, use this timezone to interpret it. Valid values are 'local', 'utc', 'z' or a valid timezone name.")

func main() {
	flag.Parse()

	count := 0
	timezoneIfUnset := time.Local
	if *timezoneFlag != "" {
		var err error
		timezoneIfUnset, err = cli.ParseTimezone(*timezoneFlag)
		cli.NoError(err, "invalid timezone provided")
	}

	scanner := cli.NewFlagArgumentScanner()
	for element, ok := scanner.ScanArgument(); ok; element, ok = scanner.ScanArgument() {
		fmt.Println(toDate(element, timezoneIfUnset))

		count++
	}

	if count == 0 {
		fmt.Println(formatDate(time.Now()))
	}
}

func toDate(element string, timezoneIfUnset *time.Location) (out string) {
	if location, found := cli.GetTimeZoneAbbreviationLocation(element); found {
		// There is just a location, gives the current time in that location
		return formatDateAt(time.Now().In(location))
	}

	hint := cli.DateLikeHintNone
	switch {
	case *asUnixMillisFlag:
		hint = cli.DateLikeHintUnixMilliseconds
	case *asUnixSecondsFlag:
		hint = cli.DateLikeHintUnixSeconds
	}

	if *asUnixSecondsFlag {
		hint = cli.DateLikeHintUnixSeconds
	}

	parsed, _, ok := cli.ParseDateLikeInput(element, hint, timezoneIfUnset)
	if !ok {
		return fmt.Sprintf("Unable to interpret %q", element)
	}

	return formatDate(parsed)
}

func formatDate(in time.Time) string {
	return fmt.Sprintf("%s (%s)", in.Local().Format(time.RFC3339), in.UTC().Format(time.RFC3339))
}

func formatDateAt(in time.Time) string {
	return fmt.Sprintf("%s (%s)", in.Format(time.RFC3339), in.UTC().Format(time.RFC3339))
}
