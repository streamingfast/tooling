package main

import (
	"flag"
	"fmt"
	"strconv"

	"github.com/dfuse-io/tooling/cli"
)

var asBinary = flag.Bool("b", false, "Use IEC base 2 representation for bytes, i.e. KB = 1024, MB = 1024^2, etc.")
var asInternational = flag.Bool("si", false, "Use International System of Units (SI) base 10 representation for bytes, i.e. KB = 1000, MB = 1000^2, etc.")

func main() {
	flag.Parse()

	scanner := cli.NewArgumentScanner()
	for element, ok := scanner.ScanArgument(); ok; element, ok = scanner.ScanArgument() {
		fmt.Println(humanize(element))
	}
}

func humanize(element string) string {

	if cli.DecRegexp.MatchString(element) {
		value, err := strconv.ParseInt(element, 10, 64)
		cli.NoError(err, "invalid dec value %q", element)

		return humanizeBytes(value)
	}

	if cli.HexRegexp.MatchString(element) {
		value, err := strconv.ParseInt(element, 16, 64)
		cli.NoError(err, "invalid hex value %q", element)

		return humanizeBytes(value)
	}

	return element
}

func humanizeBytes(value int64) string {
	metricSystem := base10MetricSystem
	if *asBinary {
		metricSystem = base2MetricSystem
	}

	for _, entry := range metricSystem {
		if value >= entry.InBytes {
			return format(value, entry)
		}
	}

	return format(value, metricSystem[len(metricSystem)-1])
}

func format(value int64, entry metricSystemEntry) string {
	converted := float64(value) / float64(entry.InBytes)

	return fmt.Sprintf("%.2f %s", converted, entry.Unit)
}

var base10MetricSystem = []metricSystemEntry{
	{Unit: "TB", InBytes: 1000 * 1000 * 1000 * 1000},
	{Unit: "GB", InBytes: 1000 * 1000 * 1000},
	{Unit: "MB", InBytes: 1000 * 1000},
	{Unit: "KB", InBytes: 1000},
	{Unit: "bytes", InBytes: 1},
}

var base2MetricSystem = []metricSystemEntry{
	{Unit: "TiB", InBytes: 1024 * 1024 * 1024 * 1024},
	{Unit: "GiB", InBytes: 1024 * 1024 * 1024},
	{Unit: "MiB", InBytes: 1024 * 1024},
	{Unit: "KiB", InBytes: 1024},
	{Unit: "bytes", InBytes: 1},
}

type metricSystemEntry struct {
	Unit    string
	InBytes int64
}

//go:generate go-enum -f=$GOFILE --marshal --names

//
// ENUM(
//   Byte
//   KiloByte
//   MegaByte
//   GigaByte
//   TeraByte
// )
type Size uint
