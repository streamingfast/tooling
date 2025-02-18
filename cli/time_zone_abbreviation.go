package cli

import (
	"bufio"
	_ "embed"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed time_zone_abbreviation.csv
var timeZoneAbbreviationCSV string

var timeZoneAbbreviationToOffsetSecondsFunc = sync.OnceValue(parseTimeZoneAbbreviationLocation)

func GetTimeZoneAbbreviationLocation(in string) (*time.Location, bool) {
	zoneAbbreviationToLocation := timeZoneAbbreviationToOffsetSecondsFunc()

	location, exists := zoneAbbreviationToLocation[in]
	return location, exists
}

func parseTimeZoneAbbreviationLocation() map[string]*time.Location {
	bestOffsets := make(map[string]int)

	scanner := bufio.NewScanner(strings.NewReader(timeZoneAbbreviationCSV))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			continue
		}

		abbrev := strings.TrimSpace(parts[0])
		offset, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			continue
		}

		// Compare with -05:00 (i.e. -18000 seconds)
		diff := math.Abs(float64(offset + 18000))
		if existing, ok := bestOffsets[abbrev]; ok {
			existingDiff := math.Abs(float64(existing + 18000))
			if diff < existingDiff {
				bestOffsets[abbrev] = offset
			}
		} else {
			bestOffsets[abbrev] = offset
		}
	}

	zoneToLocationMap := make(map[string]*time.Location)
	for abbrev, offset := range bestOffsets {
		zoneToLocationMap[abbrev] = time.FixedZone(abbrev, offset)
	}

	return zoneToLocationMap
}
