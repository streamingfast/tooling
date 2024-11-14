package main

import (
	"fmt"
	"regexp"

	"github.com/spf13/cobra"
	. "github.com/streamingfast/cli"
	"github.com/streamingfast/logging"
	"github.com/streamingfast/tooling/cli"
)

var version = "dev"
var zlog, _ = logging.PackageLogger("ansi_strip", "github.com/streamingfast/tooling/cmd/ansi_strip")

func main() {
	Run(
		"ansi_strip",
		"Remove ANSI escape codes from the input(s)",
		Description(`
			Takes a string that has ANSI escape codes for coloring and formatting and removes them.

			Taking an input of the form:
				
				[90m2:40PM[0m [32mINF[0m finalized block

			Will turn it into:

			  	2:40PM INF finalized block
		`),
		Example(`
			# Decode the following input file
			cat test.txt | ansi_strip

			# Decode the following command's output
			echo -e "\033[90m2:40PM\033[0m \033[32mINF\033[0m finalized block" | ansi_strip
		`),
		ArbitraryArgs(),

		ConfigureVersion(version),
		ConfigureViper("ANSI_STRIP"),
		OnCommandErrorLogAndExit(zlog),

		Execute(execute),
	)
}

// Credits to https://github.com/acarl005/stripansi/blob/master/stripansi.go#L7 for the regex
var re = regexp.MustCompile("[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))")

func execute(cmd *cobra.Command, args []string) error {
	scanner := cli.NewArgumentScanner(args)
	for element, ok := scanner.ScanArgument(); ok; element, ok = scanner.ScanArgument() {
		fmt.Println(re.ReplaceAllString(element, ""))
	}

	return nil
}
