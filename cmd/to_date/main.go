package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/streamingfast/tooling/cli"
)

var asUnixSeconds = flag.Bool("s", false, "Avoid heuristics based to determine decimal range value and assume it's UNIX seconds since epoch")
var asUnixMillis = flag.Bool("ms", false, "Avoid heuristics based to determine decimal range value and assume it's UNIX milliseconds since epoch")

func main() {
	flag.Parse()

	count := 0

	scanner := cli.NewFlagArgumentScanner()
	for element, ok := scanner.ScanArgument(); ok; element, ok = scanner.ScanArgument() {
		fmt.Println(toDate(element))

		count++
	}

	if count == 0 {
		fmt.Println(formatDate(time.Now()))
	}
}

var _, localOffset = time.Now().Zone()

func toDate(element string) (out string) {
	hint := cli.DateLikeHintNone
	switch {
	case *asUnixMillis:
		hint = cli.DateLikeHintUnixMilliseconds
	case *asUnixSeconds:
		hint = cli.DateLikeHintUnixSeconds
	}

	if *asUnixSeconds {
		hint = cli.DateLikeHintUnixSeconds
	}

	parsed, _, ok := cli.ParseDateLikeInput(element, hint)
	if !ok {
		return fmt.Sprintf("Unable to interpret %q", element)
	}

	return formatDate(parsed)
}

func formatDate(in time.Time) string {
	return fmt.Sprintf("%s (%s)", in.Local().Format(time.RFC3339), in.UTC().Format(time.RFC3339))
}
