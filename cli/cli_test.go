package cli

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_fromLayouts(t *testing.T) {
	type want struct {
		out        time.Time
		parsedFrom DateParsedFrom
		ok         bool
	}
	tests := []struct {
		element string
		want    want
	}{
		{"2023-04-13T14:25:27.180-0400", want{time.Unix(1681410327, 180000000), DateParsedFromLayout, true}},
		{"Wed Aug 09 2023 22:02:05 GMT-0400", want{time.Unix(1691647325, 0), DateParsedFromLayout, true}},
	}
	for _, tt := range tests {
		t.Run(tt.element, func(t *testing.T) {
			gotOut, gotParsedFrom, gotOk := fromLayouts(tt.element)

			assert.Equal(t, tt.want.out, gotOut, "Which equals to time.Unix(%d, %d)", gotOut.UnixNano()/1e9, (gotOut.UnixNano() % 1e9))
			assert.Equal(t, tt.want.parsedFrom, gotParsedFrom)
			assert.Equal(t, tt.want.ok, gotOk)
		})
	}
}
