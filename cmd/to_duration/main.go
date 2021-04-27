package main

import (
	"flag"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dfuse-io/tooling/cli"
)

var maybeDurationRegex = regexp.MustCompile("^[-0-9\\.( |h|m|s|ms|µs|ns)]+$")

var asNanoseconds = flag.Bool("ns", false, "Decode the value as having nanosecond unit")
var asMicroseconds = flag.Bool("us", false, "Decode the value as having microsecond unit")
var asMillisecondsFlag = flag.Bool("ms", false, "Decode the value as having millisecond unit")
var asSecondsFlag = flag.Bool("s", false, "Decode the value as having second unit")
var asMinutesFlag = flag.Bool("m", false, "Decode the value as having minute unit")
var asHoursFlag = flag.Bool("h", false, "Decode the value as having hour unit")

func main() {
	flag.Parse()

	var unit time.Duration
	switch {
	case *asNanoseconds:
		unit = time.Nanosecond
	case *asMicroseconds:
		unit = time.Nanosecond
	case *asMillisecondsFlag:
		unit = time.Millisecond
	case *asSecondsFlag:
		unit = time.Second
	case *asMinutesFlag:
		unit = time.Minute
	case *asHoursFlag:
		unit = time.Hour
	default:
		cli.Quit("Exclusively one of the unit flag -ns (Nanosecond), -us (Microseond), -ms (Millisecond), -s (Second), -m (Minute) or -h (Hour) must be provided")
	}

	scanner := cli.NewArgumentScanner()
	for element, ok := scanner.ScanArgument(); ok; element, ok = scanner.ScanArgument() {
		fmt.Println(toDuration(element, unit))
	}
}

func toDuration(element string, unit time.Duration) string {
	if element == "" {
		return ""
	}

	if cli.DecRegexp.MatchString(element) {
		value, _ := strconv.ParseInt(element, 10, 64)

		return durationToString(time.Duration(value) * unit)
	}

	if maybeDurationRegex.MatchString(element) {
		parsed, err := time.ParseDuration(strings.ReplaceAll(element, " ", ""))
		if err == nil {
			return durationToUnit(parsed, unit)
		}

		// There is an error, unable to parse element as a time.Duration, ignore it
	}

	return element
}

func durationToUnit(d time.Duration, unit time.Duration) string {
	switch unit {
	case time.Nanosecond:
		return strconv.FormatInt(d.Nanoseconds(), 10)
	case time.Microsecond:
		usec := d / time.Microsecond
		nusec := d % time.Microsecond

		return strconv.FormatFloat(float64(usec)+float64(nusec)/1e3, 'f', -1, 64)
	case time.Millisecond:
		msec := d / time.Millisecond
		nmsec := d % time.Millisecond

		return strconv.FormatFloat(float64(msec)+float64(nmsec)/1e6, 'f', -1, 64)
	case time.Second:
		return strconv.FormatFloat(d.Seconds(), 'f', -1, 64)
	case time.Minute:
		return strconv.FormatFloat(d.Minutes(), 'f', -1, 64)
	case time.Hour:
		return strconv.FormatFloat(d.Hours(), 'f', -1, 64)
	default:
		panic(fmt.Errorf("invalid unit %s, should have matched one of the pre-defined unit", unit))
	}
}

// durationToString is a copy of time.Duration.String() to add spaces between components
// for easier readability.
func durationToString(d time.Duration) string {
	// Largest time is 2540400h 10m 10.000000000s
	var buf [34]byte
	w := len(buf)

	u := uint64(d)
	neg := d < 0
	if neg {
		u = -u
	}

	if u < uint64(time.Second) {
		// Special case: if duration is smaller than a second,
		// use smaller units, like 1.2ms
		var prec int
		w--
		buf[w] = 's'
		w--
		switch {
		case u == 0:
			return "0s"
		case u < uint64(time.Microsecond):
			// print nanoseconds
			prec = 0
			buf[w] = 'n'
		case u < uint64(time.Millisecond):
			// print microseconds
			prec = 3
			// U+00B5 'µ' micro sign == 0xC2 0xB5
			w-- // Need room for two bytes.
			copy(buf[w:], "µ")
		default:
			// print milliseconds
			prec = 6
			buf[w] = 'm'
		}
		w, u = fmtFrac(buf[:w], u, prec)
		w = fmtInt(buf[:w], u)
	} else {
		w--
		buf[w] = 's'

		w, u = fmtFrac(buf[:w], u, 9)

		// u is now integer seconds
		w = fmtInt(buf[:w], u%60)
		u /= 60

		// u is now integer minutes
		if u > 0 {
			w--
			w--
			copy(buf[w:], "m ")
			w = fmtInt(buf[:w], u%60)
			u /= 60

			// u is now integer hours
			// Stop at hours because days can be different lengths.
			if u > 0 {
				w--
				w--
				copy(buf[w:], "h ")
				w = fmtInt(buf[:w], u)
			}
		}
	}

	if neg {
		w--
		buf[w] = '-'
	}

	return string(buf[w:])
}

// fmtFrac formats the fraction of v/10**prec (e.g., ".12345") into the
// tail of buf, omitting trailing zeros. It omits the decimal
// point too when the fraction is 0. It returns the index where the
// output bytes begin and the value v/10**prec.
func fmtFrac(buf []byte, v uint64, prec int) (nw int, nv uint64) {
	// Omit trailing zeros up to and including decimal point.
	w := len(buf)
	print := false
	for i := 0; i < prec; i++ {
		digit := v % 10
		print = print || digit != 0
		if print {
			w--
			buf[w] = byte(digit) + '0'
		}
		v /= 10
	}
	if print {
		w--
		buf[w] = '.'
	}
	return w, v
}

// fmtInt formats v into the tail of buf.
// It returns the index where the output begins.
func fmtInt(buf []byte, v uint64) int {
	w := len(buf)
	if v == 0 {
		w--
		buf[w] = '0'
	} else {
		for v > 0 {
			w--
			buf[w] = byte(v%10) + '0'
			v /= 10
		}
	}
	return w
}
