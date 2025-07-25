package main

import (
	"flag"
	"fmt"
	"math/big"
	"strings"

	"github.com/streamingfast/tooling/cli"
)

var asBinary = flag.Bool("b", false, "Use IEC base 2 representation for bytes, i.e. KiB = 1024, MiB = 1024^2, etc.")
var asInternational = flag.Bool("si", false, "Use International System of Units (SI) base 10 representation for bytes, i.e. KB = 1000, MB = 1000^2, etc.")
var compact = flag.Bool("c", false, "Compact output giving only one of -b or -si, depending on which one is used.")

func main() {
	flag.Parse()

	cli.Ensure(!(*asBinary && *asInternational), "You cannot use both -b and -si flags at the same time")

	scanner := cli.NewFlagArgumentScanner()
	for element, ok := scanner.ScanArgument(); ok; element, ok = scanner.ScanArgument() {
		fmt.Println(humanize(element))
	}
}

func humanize(element string) string {
	if cli.DecRegexp.MatchString(element) {
		value, ok := new(big.Int).SetString(element, 10)
		cli.Ensure(ok, "invalid decimal value %q", element)

		return humanizeBytes(value)
	}

	if cli.HexRegexp.MatchString(element) {
		value, ok := new(big.Int).SetString(strings.TrimPrefix(strings.ToLower(element), "0x"), 16)
		cli.Ensure(ok, "invalid hex value %q", element)

		return humanizeBytes(value)
	}

	return element
}

func humanizeBytes(value *big.Int) string {
	inInternational := formatBiggestValue(value, base10MetricSystem)
	inBinary := formatBiggestValue(value, base2MetricSystem)

	if *compact {
		if *asInternational {
			return inInternational
		}

		return inBinary
	}

	return fmt.Sprintf("%s (%s)", inBinary, inInternational)
}

func formatBiggestValue(value *big.Int, metricSystem []metricSystemEntry) string {
	for _, entry := range metricSystem {
		if value.Cmp(entry.InBytes) >= 0 {
			return format(value, entry)
		}
	}

	return format(value, metricSystem[len(metricSystem)-1])
}

func format(value *big.Int, entry metricSystemEntry) string {
	converted := new(big.Rat).SetFrac(value, entry.InBytes)
	return fmt.Sprintf("%s %s", converted.FloatString(2), entry.Unit)
}

var base10MetricSystem = []metricSystemEntry{
	{Unit: "PB", InBytes: big.NewInt(1000 * 1000 * 1000 * 1000 * 1000)},
	{Unit: "TB", InBytes: big.NewInt(1000 * 1000 * 1000 * 1000)},
	{Unit: "GB", InBytes: big.NewInt(1000 * 1000 * 1000)},
	{Unit: "MB", InBytes: big.NewInt(1000 * 1000)},
	{Unit: "KB", InBytes: big.NewInt(1000)},
	{Unit: "bytes", InBytes: big.NewInt(1)},
}

var base2MetricSystem = []metricSystemEntry{
	{Unit: "PiB", InBytes: big.NewInt(1024 * 1024 * 1024 * 1024 * 1024)},
	{Unit: "TiB", InBytes: big.NewInt(1024 * 1024 * 1024 * 1024)},
	{Unit: "GiB", InBytes: big.NewInt(1024 * 1024 * 1024)},
	{Unit: "MiB", InBytes: big.NewInt(1024 * 1024)},
	{Unit: "KiB", InBytes: big.NewInt(1024)},
	{Unit: "bytes", InBytes: big.NewInt(1)},
}

type metricSystemEntry struct {
	Unit    string
	InBytes *big.Int
}
