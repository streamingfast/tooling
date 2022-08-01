package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	. "github.com/streamingfast/cli"
	"github.com/streamingfast/tooling/cli"
)

func main() {
	Run(
		"re_string",
		"Turns a string containing literal '\n' and '\t' into their character representation",
		Description(`
			The goal of this command is to "decode" raw string escape sequences into their
			actual representation so in essence, it's kind of a reformat of a "raw" string.
		`),
		Example(`
			# Would print
			#
			#   first string
			#       second string
			#
			re_string "first string\n\tsecond string"

			# Works also with piping
			pbpaste | re_string
		`),
		Execute(func(_ *cobra.Command, args []string) error {
			scanner := cli.NewArgumentScanner(args)
			for element, ok := scanner.ScanArgument(); ok; element, ok = scanner.ScanArgument() {
				fmt.Println(reString(element))
			}

			return nil
		}),
	)
}

func reString(in string) string {
	in = strings.ReplaceAll(in, "\\n", "\n")
	in = strings.ReplaceAll(in, "\\t", "\t")

	return in
}
