package main

import (
	"flag"
	"fmt"
	"math"
	"sort"
	"strconv"

	"github.com/dfuse-io/tooling/cli"
)

var unit = flag.String("u", "", "An optional unit value, appended verbatim to each element of the final report if present")

func main() {
	flag.Parse()

	elementCount := uint64(0)
	sum := 0.0
	var min, max *float64
	var distribution []float64

	scanner := cli.NewArgumentScanner()
	for element, ok := scanner.ScanArgument(); ok; element, ok = scanner.ScanArgument() {
		value := parse(element)
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

	fmt.Printf("Count: %d\n", count(elementCount))
	fmt.Printf("Range: Min %s - Max %s\n", float(*min), float(*max))
	fmt.Printf("Sum: %s\n", float(sum))
	fmt.Printf("Average: %s\n", float(sum/float64(elementCount)))
	fmt.Printf("Median: %s\n", float(median(distribution)))
	fmt.Printf("Standard Deviation: %s\n", float(standardDeviation(sum/float64(elementCount), distribution)))
}

func parse(element string) float64 {
	value, err := strconv.ParseFloat(element, 64)
	cli.NoError(err, "all arguments should be a number, %q wasn't", element)

	return value
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

type count uint64

func (c count) String() string {
	value := strconv.FormatUint(uint64(c), 64)
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
