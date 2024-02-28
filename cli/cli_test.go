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
			want{dateIn(t, "2024-02-28 14:52:41.388325098Z", weirdEmptyLocation), DateParsedFromLayout, true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.element, func(t *testing.T) {
			gotOut, gotParsedFrom, gotOk := ParseDateLikeInput(tt.element, DateLikeHintNone, testLocation)

			// Bool first for easier spotting that parsing failed
			assert.Equal(t, tt.want.ok, gotOk, "Date %q parsing failed, no layouts were able to parse this date", tt.element)

			assert.Equal(t, tt.want.out, gotOut, "Which equals to date(t, %q)", gotOut.Format(testLayout))
			assert.Equal(t, tt.want.parsedFrom, gotParsedFrom)
		})
	}
}

var testLayout = "2006-01-02 15:04:05.999999999Z07:00"

// It seems location of "2024-02-28 14:52:41.388325098 +0000 UTC" when parsed with
// Golang time layout "2006-01-02 15:04:05.999999999 -0700 UTC" gives a location
// that prints as `time.Location("")` but creating an empty location with
// &time.Location{} does equal to the location yielded.
//
// So for the test to pass, we need to get this "location" from the same computation
// and use it in the test assertion
var weirdEmptyLocation = computeWeirdEmptyLocation()

func date(t *testing.T, in string) time.Time {
	t.Helper()
	tm, err := time.Parse(testLayout, in)
	require.NoError(t, err)
	return tm
}

func dateIn(t *testing.T, in string, loc *time.Location) time.Time {
	t.Helper()
	return date(t, in).In(loc)
}

func dateInTestLocation(t *testing.T, in string) time.Time {
	t.Helper()
	return dateIn(t, in, testLocation)
}

func computeWeirdEmptyLocation() *time.Location {
	out, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 UTC", "2024-02-28 14:52:41.388325098 +0000 UTC")
	if err != nil {
		panic(err)
	}

	return out.Location()
}
