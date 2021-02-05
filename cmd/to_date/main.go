package main

import (
	"flag"
	"fmt"
	"strconv"
	"time"

	"github.com/dfuse-io/tooling/cli"
)

var asUnixSeconds = flag.Bool("s", false, "Avoid heuristics based to determine decimal range value and assume it's UNIX seconds since epoch")
var asUnixMillis = flag.Bool("m", false, "Avoid heuristics based to determine decimal range value and assume it's UNIX milliseconds since epoch")

func main() {
	flag.Parse()

	scanner := cli.NewArgumentScanner()
	for element, ok := scanner.ScanArgument(); ok; element, ok = scanner.ScanArgument() {
		fmt.Println(toDate(element))
	}
}

func toDate(element string) (out string) {
	dateTime := time.Time{}
	defer func() {
		if !dateTime.IsZero() {
			out = formatDate(dateTime)
		}
	}()

	if element == "" {
		return
	}

	if element == "now" {
		dateTime = time.Now()
		return
	}

	if cli.DecRegexp.MatchString(element) {
		value, _ := strconv.ParseUint(element, 10, 64)

		if *asUnixMillis {
			dateTime = fromUnixMilliseconds(value)
			return
		}

		if *asUnixSeconds {
			dateTime = fromUnixSeconds(value)
			return
		}

		// If the value is lower than this Unix seconds timestamp representing 3000-01-01, we assume it's a Unix seconds value
		if value <= 32503683661 {
			dateTime = fromUnixSeconds(value)
			return
		}

		// In all other cases, we assume it's a Unix milliseconds
		dateTime = fromUnixMilliseconds(value)
		return
	}

	// Try all layouts we support
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, element)
		if err == nil {
			dateTime = parsed
			return
		}
	}

	return
}

func formatDate(in time.Time) string {
	return fmt.Sprintf("%s (%s)", in.Local().Format(time.RFC3339), in.Format(time.RFC3339))
}

func fromUnixSeconds(value uint64) time.Time {
	return time.Unix(int64(value), 0).UTC()
}

func fromUnixMilliseconds(value uint64) time.Time {
	ns := (int64(value) % 1000) * int64(time.Millisecond)

	return time.Unix(int64(value)/1000, ns).UTC()
}

var layouts = []string{
	// Sorted from most probably to less probably
	time.RFC3339,
	time.RFC3339Nano,
	time.UnixDate,
	time.RFC850,
	time.RubyDate,
	time.RFC1123,
	time.RFC1123Z,
	time.RFC822,
	time.RFC822Z,
	time.ANSIC,
	time.Stamp,
	time.StampMilli,
	time.StampMicro,
	time.StampNano,
}
