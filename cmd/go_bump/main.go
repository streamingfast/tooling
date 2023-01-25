package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/streamingfast/cli"
	. "github.com/streamingfast/cli"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

var zlog, _ = logging.ApplicationLogger("go_dump", "github.com/streamingfast/tooling/cmd/go_bump")

func main() {
	Run(
		"go_bump <id> [<id>...]",
		"Bump dependency id(s) provided with shortcut ids expanded based on user specify config file",
		Execute(run),
		MinimumNArgs(1),
		Flags(func(flags *pflag.FlagSet) {
			flags.BoolP("tidy", "t", false, "Perform a 'go mod tidy' after bumping dependencies")
		}),
		Description(`
			This command works best if you configure a config file that defines your most often used
			project. Create a config file in '$HOME/.config/go_bump/default.yaml' with the following
			content:

			  default_repo_shortcut: github.com                    # To define value of leading @
			  default_project_shortcut: github.com/streamingfast   # To define value of leading ~
			  default_branch_shortcut: "@develop"                  # To define value of trailing !

			With this config, you will be able to more easily bump dependencies for your
			project.

			You can just put an input value, in which case '<default_project_shortcut>' is
			prepended and '<default_branch_shortcut>' is appended leading for example 'bstream'
			input to perform 'go get github.com/streamingfast/bstream@develop'.

			If you have a '<default_branch_shortcut>' value set and which to update to
			latest tagged version, use '@latest' suffix.
		`),
		Example(`
			# Expands to 'go get <default_project_shortcut>@<default_branch_shortcut>' (dynamic values from config file)
			go_bump bstream

			# Expands to 'go get <default_project_shortcut>@v0.1.0' (dynamic values from config file)
			go_bump bstream@v0.1.0

			# Expands 'go get <default_repo_shortcut>/eoscanada/eos-go@<default_branch_shortcut>' (dynamic values from config file)
			go_bump @eoscanada/eos-go
		`),
	)
}

func run(cmd *cobra.Command, args []string) error {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("unable to determine use directory")
	}

	configFile := filepath.Join(userHome, ".config", "go_bump", "default.yaml")
	config, err := LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	shouldRunModTidy, provided := cli.MustGetBoolProvided(cmd, "tidy")
	if provided {
		config.AfterBump.GoModTidy = shouldRunModTidy
	}

	packageIDs := make([]PackageID, len(args))

	for i, arg := range args {
		resolved, err := UnresolvedPackageID(arg).Resolve(config)
		if err != nil {
			return fmt.Errorf("package id resolution: %w", err)
		}

		packageIDs[i] = resolved
	}

	bumpedSomething := bump(cmd.Context(), packageIDs...)

	if bumpedSomething && config.AfterBump.GoModTidy {
		runGoModTidy(cmd.Context())
	}

	return nil
}

func bump(ctx context.Context, packageIDs ...PackageID) (bumpedSomething bool) {
	args := make([]string, 1+len(packageIDs))
	args[0] = "get"

	for i, packageID := range packageIDs {
		args[i+1] = string(packageID)
	}

	cmd := exec.CommandContext(ctx, "go", args...)
	rawOutput, err := cmd.CombinedOutput()
	if err != nil {
		// FIXME: Create specialized error that formats it correctly to stderr
		printlnError("Failed to bump packages %s (command %q)", strings.Join(args[1:], ", "), cmd)
		os.Stderr.Write(rawOutput)
		os.Exit(1)
	}

	output := string(rawOutput)
	fmt.Print(output)

	zlog.Debug("completed bumping of package", zap.Strings("package_ids", args[1:]))

	// Not perfect, but should be good enough
	return strings.Contains(output, "go: upgraded")
}

func runGoModTidy(ctx context.Context) {
	cmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	output, err := cmd.CombinedOutput()
	if err != nil {
		printlnError("Failed to run 'go mod tidy'")
		os.Stderr.Write(output)
		os.Exit(1)
	}

	zlog.Debug("completed go mod tidy")
}

func printlnError(message string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, message+"\n", args...)
}

func usage() string {
	return `
Bumps dependency id(s) provided on the currently dectected Golang module,
commit the changes and create a commit with a standardize message.

The <id> format by defaults use the standard 'go get' format, however, we
provide some shortcuts:

  GitHub Shortcut               @dfuse-io/bstream@develop (equivalent to github.com/streamingfast/bstream@develop)
  Default Project Shortcut      ~bstream@develop (equivalent to github.com/streamingfast/bstream@develop)
  Default Branch Shortcut       ~bstream! (equivalent to github.com/streamingfast/bstream@develop)

You can configure the default values used for ~ and ! via config file '$HOME/config/go_bump/default.yaml'
that if exists, can override the values

  default_repo_shortcut: github.com                # To define value of leading @
  default_project_shortcut: github.com/something   # To define value of leading ~
  default_branch_shortcut: @develop                # To define value of trailing !

If it does *not* start with a ~ or @ character, it is considered a 'go get' format.
The '!' is still expanded in all situation if it terminates the id.
`
}

func silenceUsageOnError(fn func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		err := fn(cmd, args)
		if err != nil {
			cmd.SilenceUsage = true
		}

		return err
	}
}
