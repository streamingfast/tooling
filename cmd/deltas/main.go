package main

import (
	"fmt"
	"math/big"
	"time"

	"github.com/spf13/cobra"
	. "github.com/streamingfast/cli"
	"github.com/streamingfast/tooling/cli"
)

func main() {
	Run(
		"deltas <line> [<line> ...]",
		"Compute the delta between subsequent lines of output (prints it at end of read line)",
		Description(`
			For a list of received lines, compute the delta between two consecutive lines and print
			it, appending it at the end of the line.

			At least two line is required, if a single line is received, an error is emitted.

			The line can contain numbers (integers) or date as parsed using
		`),
		Example(`
			# Print the deltas from the lines from 'stdin'
			deltas

			# Print the deltas from provided values from terminal arguments
			deltas 10 12 14

		`),
		Execute(func(_ *cobra.Command, args []string) error {
			scanner := cli.NewArgumentScanner(args)

			var previous *big.Int
			var previousTimestamp time.Time

			var lineCount uint
			for element, ok := scanner.ScanArgument(); ok; element, ok = scanner.ScanArgument() {
				lineCount++

				timestamp, parsedFrom, ok := cli.ParseDateLikeInput(element, cli.DateLikeHintNone, time.Local)
				if ok && parsedFrom == cli.DateParsedFromLayout {
					if previousTimestamp.IsZero() {
						previousTimestamp = timestamp

						fmt.Printf("%s (-)\n", element)
						continue
					}

					// We had a previous element, compute the delta
					delta := timestamp.Sub(previousTimestamp)
					sign := "+"
					if delta <= 0 {
						// Sign is removed because it's either 0 or negative, if negative, the String() representation will add it
						sign = ""
					}

					fmt.Printf("%s (%s%s)\n", element, sign, delta)
					previousTimestamp = timestamp
				} else {
					number := toNumber(element)

					if previous == nil {
						previous = number

						fmt.Printf("%s (-)\n", element)
						continue
					}

					// We had a previous element, compute the delta
					delta := new(big.Int).Sub(number, previous)

					sign := "+"
					if delta.Sign() <= 0 {
						// Sign is removed because it's either 0 or negative, if negative, the `Text(10)` call is going to add it
						sign = ""
					}

					fmt.Printf("%s (%s%s)\n", element, sign, delta)
					previous = number
				}

			}

			cli.Ensure(lineCount >= 2, "At least 2 lines is required for this tool, received %d", lineCount)

			return nil
		}),
	)
}

func toNumber(element string) *big.Int {
	if element == "" {
		return big.NewInt(0)
	}

	return cli.ReadInteger(element)
}
