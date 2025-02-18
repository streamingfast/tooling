package cli

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testLocation = time.FixedZone("EST", -5*60*60)

func init() {
	timeNow = func() time.Time {
		return time.Unix(1732886907, 718000000)
	}
}

func Test_ParseDateLikeInput(t *testing.T) {
	type want struct {
		out        time.Time
		parsedFrom DateParsedFrom
		ok         bool
	}
	tests := []struct {
		element string
		want    want
	}{
		{
			"now",
			want{date(t, "2024-11-29 08:28:27.718-05:00"), DateParsedFromLayout, true},
		},
		{
			"2023-04-13T14:25:27.180-0400",
			want{date(t, "2023-04-13 14:25:27.18-04:00"), DateParsedFromLayout, true},
		},
		{
			"Wed Aug 09 2023 22:02:05 GMT-0400",
			want{date(t, "2023-08-09 22:02:05-04:00"), DateParsedFromLayout, true},
		},
		{
			"Fri Nov 10 12:07:56 2023 -0500",
			want{date(t, "2023-11-10 12:07:56-05:00"), DateParsedFromLayout, true},
		},
		{
			"11-29|08:28:27.718",
			want{dateInTestLocation(t, "2024-11-29 08:28:27.718-05:00"), DateParsedFromLayout, true},
		},
		{
			"15:30 UTC",
			want{date(t, "2024-11-29 15:30:00Z"), DateParsedFromLayout, true},
		},
		{
			// Found in Sei chain logger
			"2024-02-28 14:52:41.388325098 +0000 UTC",
			want{dateAtLocation(t, "2024-02-28 14:52:41.388325098", time.UTC), DateParsedFromLayout, true},
		},
		{
			// Found `zap-pretty` output
			"2024-07-23 14:37:10.304 EDT",
			want{date(t, "2024-07-23 14:37:10.304-04:00"), DateParsedFromLayout, true},
		},
		{
			// Found in some Telegram date reporting
			"2024-08-08 21:00:00UTC",
			want{date(t, "2024-08-08 21:00:00Z"), DateParsedFromLayout, true},
		},
		{
			// Handcrafted when needing to convert a date from timezone to local
			"2025-02-05T15:00:00 CET",
			want{dateAtLocation(t, "2025-02-05 15:00:00", cetLocation), DateParsedFromLayout, true},
		},
		{
			// Handcrafted but testing ambiguous time zone abbreviation
			"2025-02-05T15:00:00 IST",
			want{dateAtLocation(t, "2025-02-05 15:00:00", dublinLocation), DateParsedFromLayout, true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.element, func(t *testing.T) {
			gotOut, gotParsedFrom, gotOk := ParseDateLikeInput(tt.element, DateLikeHintNone, testLocation)

			// Bool first for easier spotting that parsing failed
			assert.Equal(t, tt.want.ok, gotOk, "Date %q parsing failed, no layouts were able to parse this date", tt.element)

			// time.Location is not comparable across dates usually because they are pointers and usually not the same
			// instances. Here, if the dates are not equal, we check if they are equal when ignoring the location
			// and if they are, we check if the location is the same.
			if tt.want.out != gotOut {
				if !isDateEqualWithoutLocation(t, tt.want.out, gotOut) {
					assert.Equal(t, tt.want.out, gotOut, "Which equals to date(t, %q)", gotOut.Format(testLayout))
				}

				assertEqualLocation(t, tt.want.out, gotOut)
			}

			assert.Equal(t, tt.want.parsedFrom, gotParsedFrom)
		})
	}
}

var testLayout = "2006-01-02 15:04:05.999999999Z07:00"
var testLayoutNoTimezone = "2006-01-02 15:04:05.999999999"

var cetLocation = time.FixedZone("CET", 1*60*60)
var dublinLocation = time.FixedZone("IST", 2079)

func date(t *testing.T, in string) time.Time {
	t.Helper()
	tm, err := time.Parse(testLayout, in)
	if err != nil {
		// Try without timezone, but keep original error if it fails
		var errNoTimezone error
		tm, errNoTimezone := time.Parse(testLayoutNoTimezone, in)
		if errNoTimezone == nil {
			return tm
		}
	}

	require.NoError(t, err)
	return tm
}

func dateIn(t *testing.T, in string, loc *time.Location) time.Time {
	t.Helper()
	return date(t, in).In(loc)
}

func dateAtLocation(t *testing.T, in string, loc *time.Location) time.Time {
	t.Helper()

	out := date(t, in)
	return time.Date(out.Year(), out.Month(), out.Day(), out.Hour(), out.Minute(), out.Second(), out.Nanosecond(), loc)
}

func dateInTestLocation(t *testing.T, in string) time.Time {
	t.Helper()
	return dateIn(t, in, testLocation)
}

func isDateEqualWithoutLocation(t *testing.T, expected, actual time.Time) bool {
	expectedZone, actualZoned := dateSameLocation(t, expected, actual)
	return expectedZone.Equal(actualZoned)
}

func assertEqualLocation(t *testing.T, expected, actual time.Time) {
	t.Helper()

	expectedName, expectedOffset := expected.Zone()
	actualName, actualOffset := actual.Zone()

	assert.Equal(t, expectedName, actualName, "Which equals to date(t, %q) but where location's name are different", expected.Format(testLayout))
	assert.Equal(t, expectedOffset, actualOffset, "Which equals to date(t, %q) but where location's offset are different", expected.Format(testLayout))
}

func dateSameLocation(t *testing.T, left, right time.Time) (time.Time, time.Time) {
	t.Helper()

	location := left.Location()
	leftZoned := time.Date(left.Year(), left.Month(), left.Day(), left.Hour(), left.Minute(), left.Second(), left.Nanosecond(), location)
	rightZoned := time.Date(right.Year(), right.Month(), right.Day(), right.Hour(), right.Minute(), right.Second(), right.Nanosecond(), location)

	return leftZoned, rightZoned
}
