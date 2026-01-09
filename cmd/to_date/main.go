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
		return formatDate(time.Now().In(location))
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
	local := in.Local()
	utc := in.UTC()

	_, inOffset := in.Zone()
	_, localOffset := local.Zone()
	_, utcOffset := utc.Zone()

	// Prints only current timezone + UTC version
	if inOffset == localOffset || inOffset == utcOffset {
		return fmt.Sprintf("%s (%s)", formatTime(local), formatTime(utc))
	}

	return fmt.Sprintf("%s (%s, %s)", formatTime(local), formatTime(utc), formatTime(in))
}

func formatTime(in time.Time) string {
	return in.Format(time.RFC3339)
}
