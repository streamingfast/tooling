package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	. "github.com/streamingfast/cli"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

var zlog, _ = logging.ApplicationLogger("gh_clone", "github.com/streamingfast/tooling/cmd/gh_clone")

func main() {
	Run(
		"gh_clone <id>",
		"Wraps 'git clone' to clone a GitHub repository directly into a directory <org>-<repo>",
		Execute(run),
		ExactArgs(1),
		Flags(func(flags *pflag.FlagSet) {
		}),
		Description(`
			This command is a thin wrapper around the 'git clone' command, that clones a GitHub
			repository and clones it to a directory <org>-<repo>.

			The accepted inputs are:
			- acme/example
			- github.com/acme/example
			- https://github.com/acme/example.git
			- git@github.com:acme/example.git
		`),
		Example(`
			# Executes 'git clone https://github.com/acme/example.git' into 'acme-example'
			gh_clone acme/example

			# Executes 'git clone https://github.com/acme/example.git' into 'acme-example'
			gh_clone github.com/acme/example

			# Executes 'git clone https://github.com/acme/example.git' into 'acme-example'
			gh_clone https://github.com/acme/example.git

			# Executes 'git clone git@github.com:acme/example.git' into 'acme-example'
			gh_clone git@github.com:acme/example.git
		`),
	)
}

func run(cmd *cobra.Command, args []string) error {
	resolved, err := UnresolvedInput(args[0]).Resolve(newDefaultConfig())
	if err != nil {
		return fmt.Errorf("input resolution: %w", err)
	}

	runGitClone(cmd.Context(), resolved)

	return nil
}

func runGitClone(ctx context.Context, input ResolvedInput) {
	args := []string{"clone", input.Expanded, input.Organization + "-" + input.Repository}
	cmd := exec.CommandContext(ctx, "git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		printlnError("Failed to run '%s'", cmd.String())
		os.Stderr.Write(output)
		os.Exit(1)
	}

	zlog.Debug("completed git clone command", zap.Stringer("cmd", cmd))
}

func printlnError(message string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, message+"\n", args...)
}
