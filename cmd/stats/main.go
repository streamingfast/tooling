package main

import (
	"flag"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/streamingfast/tooling/cli"
)

var unit = flag.String("u", "", "An optional unit value, appended verbatim to each element of the final report if present")

func main() {
	flag.Parse()

	elementCount := uint64(0)
	sum := 0.0
	var min, max *float64
	var distribution []float64

	scanner := cli.NewFlagArgumentScanner()
	currentValueKind := (*ValueKind)(nil)

	for element, ok := scanner.ScanArgument(); ok; element, ok = scanner.ScanArgument() {
		value, valueKind := parse(element, currentValueKind)
		if currentValueKind == nil {
			currentValueKind = &valueKind
		} else if *currentValueKind != valueKind {
			cli.Quit("All arguments should be of the same kind, %s and %s are not", *currentValueKind, valueKind)
		}

		distribution = append(distribution, value)

		elementCount++
		sum += value
		if min == nil || *min > value {
			min = &value
		}

		if max == nil || *max < value {
			max = &value
		}
	}

	if elementCount == 0 {
		fmt.Println("Statistics unavailable, no data")
		return
	}

	sort.Float64Slice(distribution).Sort()

	if currentValueKind != nil && *currentValueKind == ValueKindDuration {
		*unit = ""

		fmt.Printf("Count: %d\n", count(elementCount))
		fmt.Printf("Range: Min %s - Max %s\n", duration(*min), duration(*max))
		fmt.Printf("Sum: %s\n", duration(sum))
		fmt.Printf("Average: %s\n", duration(sum/float64(elementCount)))
		fmt.Printf("Median: %s (p90=%s p95=%s p99=%s)\n",
			duration(percentile(distribution, 50)),
			duration(percentile(distribution, 90)),
			duration(percentile(distribution, 95)),
			duration(percentile(distribution, 99)),
		)
		fmt.Printf("Standard Deviation: %s\n", duration(standardDeviation(sum/float64(elementCount), distribution)))
	} else if currentValueKind != nil && *currentValueKind == ValueKindBytes {
		fmt.Printf("Count: %d\n", count(elementCount))
		fmt.Printf("Range: Min %s - Max %s\n", bytes(*min), bytes(*max))
		fmt.Printf("Sum: %s\n", bytes(sum))
		fmt.Printf("Average: %s\n", bytes(sum/float64(elementCount)))
		fmt.Printf("Median: %s (p90=%s p95=%s p99=%s)\n",
			bytes(percentile(distribution, 50)),
			bytes(percentile(distribution, 90)),
			bytes(percentile(distribution, 95)),
			bytes(percentile(distribution, 99)),
		)
		fmt.Printf("Standard Deviation: %s\n", bytes(standardDeviation(sum/float64(elementCount), distribution)))
	} else {
		fmt.Printf("Count: %d\n", count(elementCount))
		fmt.Printf("Range: Min %s - Max %s\n", formatIntOrFloat(*min), formatIntOrFloat(*max))
		fmt.Printf("Sum: %s\n", formatIntOrFloat(sum))
		fmt.Printf("Average: %s\n", formatIntOrFloat(sum/float64(elementCount)))
		fmt.Printf("Median: %s (p90=%s p95=%s p99=%s)\n",
			formatIntOrFloat(percentile(distribution, 50)),
			formatIntOrFloat(percentile(distribution, 90)),
			formatIntOrFloat(percentile(distribution, 95)),
			formatIntOrFloat(percentile(distribution, 99)),
		)
		fmt.Printf("Standard Deviation: %s\n", formatIntOrFloat(standardDeviation(sum/float64(elementCount), distribution)))
	}
}

var durationRegex = regexp.MustCompile(`\s*(ns|us|ms|s|m|h)\s*$`)
var spacesRegex = regexp.MustCompile(`\s+`)
var bytesRegex = regexp.MustCompile(`(?i)\s*([KMGTP]i?B|(\s+)?B)\s*$`)

type byteUnit struct {
	Unit       string
	Multiplier float64
	IsBinary   bool
}

var binaryByteUnits = []byteUnit{
	{"PiB", 1024.0 * 1024.0 * 1024.0 * 1024.0 * 1024.0, true},
	{"TiB", 1024.0 * 1024.0 * 1024.0 * 1024.0, true},
	{"GiB", 1024.0 * 1024.0 * 1024.0, true},
	{"MiB", 1024.0 * 1024.0, true},
	{"KiB", 1024.0, true},
	{"B", 1.0, true},
}

var decimalByteUnits = []byteUnit{
	{"PB", 1000.0 * 1000.0 * 1000.0 * 1000.0 * 1000.0, false},
	{"TB", 1000.0 * 1000.0 * 1000.0 * 1000.0, false},
	{"GB", 1000.0 * 1000.0 * 1000.0, false},
	{"MB", 1000.0 * 1000.0, false},
	{"KB", 1000.0, false},
	{"B", 1.0, false},
}

var byteUnitMap = map[string]byteUnit{
	"pib": binaryByteUnits[0], "PIB": binaryByteUnits[0],
	"tib": binaryByteUnits[1], "TIB": binaryByteUnits[1],
	"gib": binaryByteUnits[2], "GIB": binaryByteUnits[2],
	"mib": binaryByteUnits[3], "MIB": binaryByteUnits[3],
	"kib": binaryByteUnits[4], "KIB": binaryByteUnits[4],
	"pb": decimalByteUnits[0], "PB": decimalByteUnits[0],
	"tb": decimalByteUnits[1], "TB": decimalByteUnits[1],
	"gb": decimalByteUnits[2], "GB": decimalByteUnits[2],
	"mb": decimalByteUnits[3], "MB": decimalByteUnits[3],
	"kb": decimalByteUnits[4], "KB": decimalByteUnits[4],
	"b": {Unit: "B", Multiplier: 1.0, IsBinary: true},
	"B": {Unit: "B", Multiplier: 1.0, IsBinary: true},
}

var detectedBase *bool // nil = not detected, true = binary, false = decimal

//go:generate go-enum -f=$GOFILE --marshal --names

// ENUM(
//
//	Duration
//	Number
//	Bytes
//
// )
type ValueKind uint

func parse(element string, previousKind *ValueKind) (value float64, kind ValueKind) {
	// Check for bytes first
	if bytesRegex.MatchString(element) {
		value, isBinary := parseBytes(element)

		// Track detected base on first byte input
		if detectedBase == nil {
			detectedBase = &isBinary
		}

		return value, ValueKindBytes
	}

	// Check for duration
	if durationRegex.MatchString(element) {
		// We have a duration like, let's treat is as such
		duration, err := time.ParseDuration(spacesRegex.ReplaceAllLiteralString(element, ""))
		cli.NoError(err, "Couldn't parse duration like argument %q", element)

		return float64(duration), ValueKindDuration
	}

	// Plain number
	value, err := strconv.ParseFloat(element, 64)
	cli.NoError(err, "all arguments should be a number, %q wasn't", element)

	// If previous kind was bytes, treat plain numbers as raw bytes
	if previousKind != nil && *previousKind == ValueKindBytes {
		return value, ValueKindBytes
	}

	return value, ValueKindNumber
}

func parseBytes(element string) (bytes float64, isBinary bool) {
	element = strings.TrimSpace(element)

	matches := bytesRegex.FindStringSubmatch(element)
	if len(matches) == 0 {
		cli.Quit("Invalid byte format: %q", element)
	}

	unitStr := strings.TrimSpace(matches[1])
	valueStr := strings.TrimSpace(bytesRegex.ReplaceAllString(element, ""))

	value, err := strconv.ParseFloat(valueStr, 64)
	cli.NoError(err, "Invalid number in byte value %q", element)

	unit, found := byteUnitMap[strings.ToLower(unitStr)]
	if !found {
		unit, found = byteUnitMap[strings.ToUpper(unitStr)]
	}
	if !found {
		cli.Quit("Unknown byte unit: %q", unitStr)
	}

	return value * unit.Multiplier, unit.IsBinary
}

func standardDeviation(mean float64, distribution []float64) float64 {
	sumSquaredDiffToMean := 0.0
	for _, value := range distribution {
		sumSquaredDiffToMean += math.Pow(mean-value, 2)
	}

	if sumSquaredDiffToMean == 0 {
		return 0.0
	}

	return math.Sqrt(sumSquaredDiffToMean / float64(len(distribution)-1))
}

func percentile(distribution []float64, p float64) float64 {
	if len(distribution) == 0 {
		return 0
	}
	if len(distribution) == 1 {
		return distribution[0]
	}

	rank := (p / 100.0) * float64(len(distribution)-1)
	lowerIndex := int(math.Floor(rank))
	upperIndex := int(math.Ceil(rank))

	if lowerIndex == upperIndex {
		return distribution[lowerIndex]
	}

	// Linear interpolation between the two nearest values
	weight := rank - float64(lowerIndex)
	return distribution[lowerIndex]*(1-weight) + distribution[upperIndex]*weight
}

type count uint64

func (c count) String() string {
	value := strconv.FormatUint(uint64(c), 10)
	if *unit == "" {
		return value
	}

	return value + *unit
}

type float float64

func (f float) String() string {
	value := strconv.FormatFloat(float64(f), 'f', 3, 64)
	if *unit == "" {
		return value
	}

	return value + *unit
}

type integer float64

func (i integer) String() string {
	value := strconv.FormatInt(int64(i), 10)
	if *unit == "" {
		return value
	}

	return value + *unit
}

func formatIntOrFloat(value float64) string {
	if value == float64(int64(value)) {
		return integer(value).String()
	}
	return float(value).String()
}

type duration float64

func (f duration) String() string {
	return time.Duration(f).String()
}

type bytes float64

func (b bytes) String() string {
	return formatBytes(float64(b), *unit)
}

func formatBytes(value float64, unitFlag string) string {
	// Case 1: --unit flag is a byte unit
	if isByteUnit(unitFlag) {
		return formatBytesAsUnit(value, unitFlag)
	}

	// Case 2: --unit flag is NOT a byte unit
	if unitFlag != "" {
		return fmt.Sprintf("%.0f %s", value, unitFlag)
	}

	// Case 3: --unit flag is empty - humanize
	return humanizeBytesWithBase(value)
}

func isByteUnit(unit string) bool {
	if unit == "" {
		return false
	}
	_, found := byteUnitMap[strings.ToLower(unit)]
	if !found {
		_, found = byteUnitMap[strings.ToUpper(unit)]
	}
	return found
}

func formatBytesAsUnit(bytes float64, targetUnit string) string {
	unit, found := byteUnitMap[strings.ToLower(targetUnit)]
	if !found {
		unit, found = byteUnitMap[strings.ToUpper(targetUnit)]
	}
	if !found {
		return fmt.Sprintf("%.2f %s", bytes, targetUnit)
	}

	converted := bytes / unit.Multiplier
	return fmt.Sprintf("%.2f %s", converted, unit.Unit)
}

func humanizeBytesWithBase(bytes float64) string {
	// Use detected base, default to binary
	useBinary := true
	if detectedBase != nil {
		useBinary = *detectedBase
	}

	var units []byteUnit
	if useBinary {
		units = binaryByteUnits
	} else {
		units = decimalByteUnits
	}

	// Find largest unit that fits
	for _, unit := range units {
		if bytes >= unit.Multiplier {
			return fmt.Sprintf("%.2f %s", bytes/unit.Multiplier, unit.Unit)
		}
	}

	// Fallback to plain bytes
	return fmt.Sprintf("%.0f B", bytes)
}
