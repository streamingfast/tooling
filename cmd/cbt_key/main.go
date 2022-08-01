package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	. "github.com/streamingfast/cli"
	"github.com/streamingfast/tooling/cli"
)

func main() {
	Run(
		"cbt_key",
		"Turns an hexadecimal input into cbt raw bytes key formatted like $'\x01\x02'",
		Description(`
			The goal of this command is to make it simpler when using 'cbt read prefix=<key>'
			to input hexadecimal key directly
		`),
		Example(`
			# Would prints $'\x01\xa0\x05'
			cbt_key 01a005
		`),
		Execute(func(_ *cobra.Command, args []string) error {
			scanner := cli.NewArgumentScanner(args)
			for element, ok := scanner.ScanArgument(); ok; element, ok = scanner.ScanArgument() {
				fmt.Print(cbtKey(element))
			}

			return nil
		}),
	)
}

var hexRegex = regexp.MustCompile("^(0x)?[a-fA-F0-9]+$")

func cbtKey(in string) string {
	if !hexRegex.MatchString(in) {
		return in
	}

	bytes, _ := cli.DecodeHex(in)
	elements := make([]string, len(bytes))
	for i, byteValue := range bytes {
		elements[i] = "\\x" + cli.EncodeHex([]byte{byteValue})
	}

	return fmt.Sprintf("$'%s'", strings.Join(elements, ""))
}
