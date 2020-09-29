package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/dfuse-io/tooling/cli"
	"go.uber.org/zap"
)

var separatorRegexp = regexp.MustCompile("(\\s|,|;)")

// From a line file in the format
//
// ```
// <date> <count>
// ...
// ```
//
// This script will aggregate all elements count into their day" counterpart.
// It can fill the void between days by using the `--fill` option.
var fillHolesFlag = flag.Bool("fill", false, "Will fill holes with 0 count for missing days")

var perWeekFlag = flag.Bool("week", false, "Aggregate counts for each week")
var perDayFlag = flag.Bool("day", false, "Aggregate counts for each day")
var perHourFlag = flag.Bool("hour", false, "Aggregate counts for each hour")

var debugEnabled = false
var zlog = zap.NewNop()

type config struct {
	fillHoles bool
	isNext    func(previous, current time.Time) bool
	toNext    func(current time.Time) time.Time
	truncate  func(current time.Time) time.Time
}

func init() {
	if os.Getenv("DEBUG") != "" {
		debugEnabled = true
		zlog, _ = zap.NewDevelopment()
	}
}

func main() {
	flag.Parse()

	fi, err := os.Stdin.Stat()
	cli.NoError(err, "unable to stat stdin")

	var reader io.Reader
	if (fi.Mode() & os.ModeCharDevice) == 0 {
		reader = os.Stdin
	} else {
		cli.Ensure(flag.NArg() == 1, "You must provide filename")
		reader, err = os.Open(flag.Arg(0))
		cli.NoError(err, "unable to open file %q", flag.Arg(0))
	}

	config := newConfig()
	scanner := bufio.NewScanner(reader)
	countByPeriod := map[time.Time]int{}

	for scanner.Scan() {
		date, count := parseLine(scanner.Text())
		period := config.truncate(date)
		if debugEnabled {
			zlog.Debug("truncated date to its period", zap.Time("date", date), zap.Time("truncated", period))
		}

		countByPeriod[period] = countByPeriod[period] + count
	}

	cli.NoError(scanner.Err(), "unable to fully scan lines")

	var previous time.Time
	for _, current := range sortedTimeKeys(countByPeriod) {
		if config.fillHoles && !previous.IsZero() {
			for !config.isNext(previous, current) {
				previous = config.toNext(previous)
				fmt.Printf("%s\t%d\n", previous.Format(time.RFC3339), 0)
			}
		}

		fmt.Printf("%s\t%d\n", current.Format(time.RFC3339), countByPeriod[current])
		previous = current
	}
}

func newConfig() *config {
	fillHoles := fillHolesFlag != nil && *fillHolesFlag

	perWeek := perWeekFlag != nil && *perWeekFlag
	perDay := perDayFlag != nil && *perDayFlag
	perHour := perHourFlag != nil && *perHourFlag
	zlog.Debug("per period values",
		zap.Bool("per_week", perWeek),
		zap.Bool("per_day", perDay),
		zap.Bool("per_hour", perHour),
	)

	perCount := 0
	for _, enabled := range []bool{perWeek, perDay, perHour} {
		if enabled {
			perCount++
		}
	}

	cli.Ensure(perCount <= 1, "Only one of '--week', '--day' and '--hour' must be defined")

	if perWeek {
		zlog.Debug("using a per week config")
		return &config{
			fillHoles: fillHoles,
			truncate: func(current time.Time) time.Time {
				daysToGoBackToMonday := weekDayFromMonday(current) % 7
				onMonday := current
				if daysToGoBackToMonday > 0 {
					onMonday = current.AddDate(0, 0, -daysToGoBackToMonday)
				}

				return time.Date(onMonday.Year(), onMonday.Month(), onMonday.Day(), 0, 0, 0, 0, current.Location())
			},
			isNext: func(previous, current time.Time) bool { return current.Sub(previous).Hours()/24 <= 7 },
			toNext: func(current time.Time) time.Time { return current.AddDate(0, 0, 7) },
		}
	}

	if perHour {
		zlog.Debug("using a per hour config")

		return &config{
			fillHoles: fillHoles,
			truncate: func(current time.Time) time.Time {
				return time.Date(current.Year(), current.Month(), current.Day(), current.Hour(), 0, 0, 0, current.Location())
			},
			isNext: func(previous, current time.Time) bool { return current.Sub(previous).Hours() <= 1 },
			toNext: func(current time.Time) time.Time { return current.Add(1 * time.Hour) },
		}
	}

	// Per day
	zlog.Debug("using a per day config")
	return &config{
		fillHoles: fillHoles,
		truncate: func(current time.Time) time.Time {
			return time.Date(current.Year(), current.Month(), current.Day(), 0, 0, 0, 0, current.Location())
		},
		isNext: func(previous, current time.Time) bool { return current.Sub(previous).Hours() <= 24 },
		toNext: func(current time.Time) time.Time { return current.AddDate(0, 0, 1) },
	}
}

func weekDayFromMonday(current time.Time) int {
	weekday := int(current.Weekday()) - 1
	if weekday <= -1 {
		return 6
	}

	return weekday
}

func sortedTimeKeys(mappings map[time.Time]int) (out []time.Time) {
	out = make([]time.Time, len(mappings))

	i := 0
	for key := range mappings {
		out[i] = key
		i++
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Before(out[j])
	})

	return
}

func parseLine(line string) (date time.Time, count int) {
	parts := separatorRegexp.Split(line, 2)
	cli.Ensure(len(parts) == 2, "expected 2 elements per line, got %d", len(parts))

	var err error
	date, err = time.Parse(time.RFC3339, parts[0])
	cli.NoError(err, "unable to parse date element %q", parts[0])

	count, err = strconv.Atoi(parts[1])
	cli.NoError(err, "unable to parse count element %q", parts[1])

	return
}
