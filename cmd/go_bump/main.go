package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	. "github.com/streamingfast/cli"
	"github.com/streamingfast/cli/sflags"
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
			flags.Bool("pr", false, "Create a branch, commit the bump, and open a pull request")
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

	shouldRunModTidy, provided := sflags.MustGetBoolProvided(cmd, "tidy")
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

	prMode := sflags.MustGetBool(cmd, "pr")
	if prMode {
		return runPRMode(cmd.Context(), packageIDs, config)
	}

	output, ok := goGet(cmd.Context(), os.Stdout, packageIDs...)
	if !ok {
		os.Exit(1)
	}

	if strings.Contains(output, "go: upgraded") && config.AfterBump.GoModTidy {
		runGoModTidy(cmd.Context())
	}

	return nil
}

// goGet runs `go get` for the given package IDs and writes the combined output
// to w.  It returns the full output as a string and whether the command
// succeeded.
func goGet(ctx context.Context, w io.Writer, packageIDs ...PackageID) (output string, ok bool) {
	args := make([]string, 1+len(packageIDs))
	args[0] = "get"
	for i, id := range packageIDs {
		args[i+1] = string(id)
	}

	cmd := exec.CommandContext(ctx, "go", args...)
	rawOutput, err := cmd.CombinedOutput()
	output = string(rawOutput)

	if w != nil {
		fmt.Fprint(w, output)
	}

	if err != nil {
		printlnError("Failed to bump packages %s (command %q)", strings.Join(args[1:], ", "), cmd)
		return output, false
	}

	zlog.Debug("completed bumping of package", zap.Strings("package_ids", args[1:]))
	return output, true
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
