package main

import (
	"github.com/spf13/pflag"
	. "github.com/streamingfast/cli"
)

func main() {
	Run(
		"k8x",
		"Kubernetes helper commands",
		Description(`
			A collection of helper commands for working with Kubernetes resources.
			Provides interactive prompts when arguments are missing.
		`),
		PersistentFlags(func(flags *pflag.FlagSet) {
			flags.StringP("namespace", "n", "", "Kubernetes namespace to use (if not set, uses current context)")
		}),
		SecretGroup,
	)
}
