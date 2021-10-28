package main

import (
	"flag"
	"fmt"
	"strconv"
	"time"

	"github.com/streamingfast/tooling/cli"
)

var asUnixSeconds = flag.Bool("s", false, "Format timestamp as Unix seconds")
var asUnixMillis = flag.Bool("ms", false, "Format timestamp as Unix milliseconds")
var asUnixNanos = flag.Bool("ns", false, "Format timestamp as Unix nanoseconds")

func main() {
	flag.Parse()

	scanner := cli.NewArgumentScanner()
	for element, ok := scanner.ScanArgument(); ok; element, ok = scanner.ScanArgument() {
		fmt.Println(toTimestamp(element))
	}
}

var _, localOffset = time.Now().Zone()

func toTimestamp(element string) (out string) {
	parsed, _, ok := cli.ParseDateLikeInput(element, cli.DateLikeHintNone)
	if !ok {
		return fmt.Sprintf("Unable to interpret %q", element)
	}

	return strconv.FormatInt(toTimestampValue(parsed), 10)
}

func toTimestampValue(in time.Time) int64 {
	switch {
	case *asUnixSeconds:
		return in.Unix()
	case *asUnixMillis:
		return in.UnixNano() / int64(time.Millisecond)
	case *asUnixNanos:
		return in.UnixNano()
	}

	return in.Unix()
}
