package main

import (
	"flag"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
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
		fmt.Printf("Median: %s\n", duration(median(distribution)))
		fmt.Printf("Standard Deviation: %s\n", duration(standardDeviation(sum/float64(elementCount), distribution)))
		fmt.Printf("Percentiles: p50=%s p90=%s p95=%s p99=%s\n",
			duration(percentile(distribution, 50)),
			duration(percentile(distribution, 90)),
			duration(percentile(distribution, 95)),
			duration(percentile(distribution, 99)))

	} else {
		fmt.Printf("Count: %d\n", count(elementCount))
		fmt.Printf("Range: Min %s - Max %s\n", float(*min), float(*max))
		fmt.Printf("Sum: %s\n", float(sum))
		fmt.Printf("Average: %s\n", float(sum/float64(elementCount)))
		fmt.Printf("Median: %s\n", float(median(distribution)))
		fmt.Printf("Standard Deviation: %s\n", float(standardDeviation(sum/float64(elementCount), distribution)))
		fmt.Printf("Percentiles: p50=%s p90=%s p95=%s p99=%s\n",
			float(percentile(distribution, 50)),
			float(percentile(distribution, 90)),
			float(percentile(distribution, 95)),
			float(percentile(distribution, 99)))
	}
}

var durationRegex = regexp.MustCompile(`\s*(ns|us|ms|s|m|h)\s*$`)
var spacesRegex = regexp.MustCompile(`\s+`)

//go:generate go-enum -f=$GOFILE --marshal --names

// ENUM(
//
//	Duration
//	Number
//
// )
type ValueKind uint

func parse(element string, previousKind *ValueKind) (value float64, kind ValueKind) {
	if durationRegex.MatchString(element) {
		// We have a duration like, let's treat is as such
		duration, err := time.ParseDuration(spacesRegex.ReplaceAllLiteralString(element, ""))
		cli.NoError(err, "Couldn't parse duration like argument %q", element)

		return float64(duration), ValueKindDuration
	}

	value, err := strconv.ParseFloat(element, 64)
	cli.NoError(err, "all arguments should be a number, %q wasn't", element)

	return value, ValueKindNumber
}

func median(distribution []float64) float64 {
	if len(distribution)%2 != 0 {
		return distribution[int(math.Floor(float64(len(distribution))/2.0))]
	}

	upperIndex := len(distribution) / 2
	lowerIndex := upperIndex - 1

	return (distribution[lowerIndex] + distribution[upperIndex]) / 2.0
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
	value := strconv.FormatFloat(float64(f), 'f', 5, 64)
	if *unit == "" {
		return value
	}

	return value + *unit
}

type duration float64

func (f duration) String() string {
	return time.Duration(f).String()
}
