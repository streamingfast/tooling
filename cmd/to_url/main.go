package main

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	. "github.com/streamingfast/cli"
	"github.com/streamingfast/cli/sflags"
	"github.com/streamingfast/tooling/cli"
)

var version = "dev"

func main() {
	Run(
		"to_url [-d|--decode] [-e|--encode] <input>...",
		"Encode or decode input(s) from/to URL safe format using heuristics by default to detect the way to use between decode/encode",
		Description(`
			The heuristics to decide if the inputs shall be decoded are:
			- If the input contains %, it is considered as URL encoded and will be decoded

			The heuristics to decide if the inputs shall be encoded are:
			- All other cases, e.g. input does not contain %

			You can force the process to decode or encode using the -d or -e flags.
		`),
		Flags(func(flags *pflag.FlagSet) {
			flags.BoolP("decode", "d", false, "Disables auto-detection and forces process to decode the receive input(s), mutually exclusive with -encode")
			flags.BoolP("encode", "e", false, "Disables auto-detection and forces process to encode the receive input(s), mutually exclusive with -decode")
		}),
		Example(`
			# Decode the following URL encoded string (default heuristics kicked in due to presence of %)
			to_url McGsrZDRZazAzbG9lRzFlVzUwcz0%3D

			# If you want to encode %, you need to force it
			to_url --encode Test_100%
		`),
		ArbitraryArgs(),

		ConfigureVersion(version),
		ConfigureViper("TO_URL"),

		Execute(execute),
	)
}

func execute(cmd *cobra.Command, args []string) error {
	forcedDecode := sflags.MustGetBool(cmd, "decode")
	forcedEncode := sflags.MustGetBool(cmd, "encode")

	Ensure(!(forcedEncode && forcedDecode), "Cannot use --decode and --encode at the same time")

	scanner := cli.NewArgumentScanner(args)
	for element, ok := scanner.ScanArgument(); ok; element, ok = scanner.ScanArgument() {
		fmt.Println(toUrl(element, forcedDecode, forcedEncode))
	}

	return nil
}

func toUrl(element string, forcedDecode, forcedEncode bool) string {
	if element == "" {
		return ""
	}

	if forcedDecode {
		return toDecodedUrl(element)
	}

	if forcedEncode {
		return toEncodedUrl(element)
	}

	if strings.Contains(element, "%") {
		return toDecodedUrl(element)
	}

	return element
}

func toEncodedUrl(element string) string {
	return url.QueryEscape(element)
}

func toDecodedUrl(element string) string {
	out, err := url.QueryUnescape(element)
	cli.NoError(err, "unable to decode %q as URL", element)
	return out
}
