package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_LineBuffer(t *testing.T) {
	tests := []struct {
		name      string
		newBuffer func(processor func(in string)) *lineBuffer
		inputs    []string
		outputs   []string
	}{
		{
			"passthrough empty input",
			func(processor func(string)) *lineBuffer { return newPassthroughLineBuffer(processor) },
			[]string{},
			[]string{},
		},

		{
			"passthrough single line input",
			func(processor func(string)) *lineBuffer { return newPassthroughLineBuffer(processor) },
			[]string{"line0"},
			[]string{"line0"},
		},

		{
			"passthrough multiple line input",
			func(processor func(string)) *lineBuffer { return newPassthroughLineBuffer(processor) },
			[]string{"line0", "line1", "line2"},
			[]string{"line0", "line1", "line2"},
		},

		{
			"skip 1 on single line input",
			func(processor func(string)) *lineBuffer { return newLineBuffer(1, processor) },
			[]string{"line0"},
			[]string{},
		},

		{
			"skip 2 on single line input",
			func(processor func(string)) *lineBuffer { return newLineBuffer(2, processor) },
			[]string{"line0"},
			[]string{},
		},

		{
			"skip 2 on two line input",
			func(processor func(string)) *lineBuffer { return newLineBuffer(2, processor) },
			[]string{"line0", "line1"},
			[]string{},
		},

		{
			"skip 2 on multiple line input",
			func(processor func(string)) *lineBuffer { return newLineBuffer(2, processor) },
			[]string{"line0", "line1", "line2", "line3", "line4", "line5"},
			[]string{"line0", "line1", "line2", "line3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputLines := []string{}
			processor := func(in string) { outputLines = append(outputLines, in) }
			buffer := tt.newBuffer(processor)

			for _, line := range tt.inputs {
				buffer.push(line)
			}

			assert.Equal(t, tt.outputs, outputLines)
		})
	}
}
