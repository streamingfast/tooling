package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	. "github.com/streamingfast/cli"
	"github.com/streamingfast/tooling/cli"
)

func main() {
	Run(
		"skip <count> [<file>]",
		"Skips line(s) at the beginning or the end of a line input stream",
		Description(`
			Skip is complimentary to head/tail utility, it enables you to skip line either at the begining or the end
			of a line input stream.

			Dealing with skip using head/tail is not easy and requires combination. Skip is built to skip lines and
			offer a way to skip line at the beginning of the input stream when its <count> argument is positive or
			from the end when the <count> argument is negative.

			If <file> argument is undefined, takes its input from the stdin, otherwise read <file> line by line
			and perform the skipping.
		`),
		Example(`
			# Skips the first line of input from 'stdin'
			skip 1

			# Skips the last two line of input from 'stdin' (the 0: is required otherwise -2 is seen as a flag)
			skip 0:-2

			# Skips the first line of input and the last two line input from '/tmp/lines.txt'
			skip 1:-2 /tmp/lines.txt
		`),
		RangeArgs(1, 2),
		Execute(func(_ *cobra.Command, args []string) error {
			count, err := parseSkipCount(args[0])
			NoError(err, "invalid <count> argument")

			var scanner cli.ArgumentScanner
			if len(args) == 1 {
				scanner, err = cli.NewStdinArgumentScanner()
				NoError(err, "unable to create 'stdin' scanner")
			} else {
				var closer func() error
				scanner, closer, err = cli.NewFileArgumentScanner(args[1])
				NoError(err, "unable to create 'file' scanner")

				defer closer()
			}

			lineOffset := -1
			lineProcessor := func(line string) { fmt.Println(line) }

			lineBuffer := newPassthroughLineBuffer(lineProcessor)
			if count.endSkipCount > 0 {
				lineBuffer = newLineBuffer(uint64(count.endSkipCount), lineProcessor)
			}

			for element, ok := scanner.ScanArgument(); ok; element, ok = scanner.ScanArgument() {
				lineOffset += 1

				if lineOffset >= int(count.startAt) {
					lineBuffer.push(element)
				}
			}

			return nil
		}),
	)
}

type skipCount struct {
	startAt      int64
	endSkipCount int64
}

func parseSkipCount(in string) (out skipCount, err error) {
	before, after, found := strings.Cut(in, ":")

	leftElement, err := strconv.ParseInt(before, 10, 64)
	if err != nil {
		return skipCount{}, fmt.Errorf("invalid left element in %q: %w", in, err)
	}

	var rightElement int64
	if found {
		rightElement, err = strconv.ParseInt(after, 10, 64)
		if err != nil {
			return skipCount{}, fmt.Errorf("invalid right element in %q: %w", in, err)
		}

		if leftElement < 0 {
			return skipCount{}, fmt.Errorf("invalid left element in %q: must be positive", in)
		}

		if rightElement >= 0 {
			return skipCount{}, fmt.Errorf("invalid right element in %q: must be negative and non-zero", in)
		}

		return skipCount{startAt: leftElement, endSkipCount: rightElement * -1}, nil
	}

	if leftElement < 0 {
		return skipCount{startAt: 0, endSkipCount: leftElement * -1}, nil
	}

	return skipCount{startAt: leftElement, endSkipCount: 0}, nil
}

type lineBuffer struct {
	lines         []string
	count         uint64
	slidingOffset uint64
	processor     func(line string)
}

func newPassthroughLineBuffer(processor func(line string)) *lineBuffer {
	return &lineBuffer{processor: processor}
}

func newLineBuffer(max uint64, processor func(line string)) *lineBuffer {
	return &lineBuffer{lines: make([]string, max), processor: processor}
}

func (b *lineBuffer) push(line string) {
	if b.lines == nil {
		b.processor(line)
		return
	}

	actual := b.lines[b.slidingOffset]

	b.lines[b.slidingOffset] = line
	b.moveOffset()
	b.count += 1

	if b.count > uint64(len(b.lines)) {
		b.processor(actual)
	}
}

func (b *lineBuffer) moveOffset() {
	b.slidingOffset += 1
	if cap(b.lines) == int(b.slidingOffset) {
		b.slidingOffset = 0
	}
}
