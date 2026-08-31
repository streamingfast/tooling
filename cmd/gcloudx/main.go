package main

import (
	"github.com/spf13/pflag"
	. "github.com/streamingfast/cli"
)

func main() {
	Run(
		"gcloudx",
		"Google Cloud CLI helper commands",
		Description(`
			A collection of helper commands for working with Google Cloud CLI resources.
			Provides interactive prompts when arguments are missing.
		`),
		PersistentFlags(func(flags *pflag.FlagSet) {
			flags.StringP("project", "p", "dfuseio-global", "Google Cloud project to use (if not set, uses current context)")
		}),
		SecretGroup,
	)
}
