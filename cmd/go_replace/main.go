package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/streamingfast/cli"
	. "github.com/streamingfast/cli"
	"github.com/streamingfast/cli/sflags"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

var zlog, _ = logging.ApplicationLogger("go_replace", "github.com/streamingfast/tooling/cmd/go_replace")

func main() {
	Run("go_replace <id>",
		"Replace the dependency id(s) provided on the currently detected Golang module by it's local counterpart.",
		ExactArgs(1),
		Flags(func(flags *pflag.FlagSet) {
			flags.BoolP("bump", "b", false, "Bump, using go_bump, the dependency to the latest one on drop")
			flags.BoolP("drop", "d", false, "Drop the replace statement from go.mod file")
		}),
		Description(`
			Replaced dependency id(s) provided on the current go.mod file by it's local
			or remote counterpart.

			The <id> format by defaults use the standard 'go get' format, however, we
			provide some shortcuts:

			Full Shortcut                  bstream (resolves to github.com/streamingfast/bstream when @ has value 'github.com' and ~ as value 'streamingfast')
			Platform Shortcut using @      @streamingfast/bstream (resolves to github.com/streamingfast/bstream when @ has value 'github.com')
			Default Project Shortcut       ~bstream (resolves to github.com/streamingfast/bstream when @ has value 'github.com' and ~ as value 'streamingfast')

			Where the replacement is performed to depends on the input argument. If the path
			looks like a relative path (contains either a . or a platform's path separator), then it's
			resolved relatively to current directory. Otherwise, the config value 'default_work_dir' is
			used and the input is assumed to be relative to this folder.

			The resolver uses reads the config file '$HOME/config/go_replace/default.yaml' and gets from it
			the following input:

			- 'default_work_dir' To infer local directory where dependency should be resolved to (environment variables can be used here like $HOME/work)
			- 'default_repo_shortcut' To infer platform used, defaults to 'github.com'
			- 'default_project_shortcut' To infer project used, defaults to 'github.com'

			This command can also install and verify a Git hooks that ensure you do not mistakenly
			push code that contains a local replacement.
		`),
		Example(`
			# Replace github.com/streamingfast/merger by '<default_work_dir>/merger'
			go_replace merger

			# Replace github.com/streamingfast/merger by '../merger'
			go_replace ../merger

			# Drop replacement github.com/streamingfast/merger
			go_replace -d merger

			# Install Git hook on specific <project> to ensure no local replacement is pushed
			cd <project> && go_replace hook install

			# Install Git hook on all Golang project in <projects_dir> (one that has a '.git' folder and at least one 'go.mod' file)
			cd <projects_dir> && go_replace hook install
		`),
		Execute(run),
		Group(
			"hook",
			"Git hooks to install and verify replacement (ensuring you don't push a local replacement)",
			HookInstall,
			HookVerify,
		),
	)
}

func run(cmd *cobra.Command, args []string) error {
	bump := sflags.MustGetBool(cmd, "bump")
	drop := sflags.MustGetBool(cmd, "drop")

	if bump && !drop {
		return fmt.Errorf("--bump can only be used with --drop")
	}

	userHome, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("unable to determine use directory")
	}

	configFile := filepath.Join(userHome, ".config", "go_replace", "default.yaml")
	config, err := LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	replacement, err := UnresolvedReplacement(args[0]).Resolve(config)
	if err != nil {
		return fmt.Errorf("replacement resolution: %w", err)
	}

	editDependency(cmd.Context(), args[0], replacement, drop, bump)

	return nil
}

func editDependency(ctx context.Context, input string, replacement Replacement, drop bool, bump bool) {
	cmd := exec.CommandContext(ctx, "go", "mod", "edit", "-replace", fmt.Sprintf("%s=%s", replacement.From, replacement.To))
	if drop {
		cmd = exec.CommandContext(ctx, "go", "mod", "edit", "-dropreplace", replacement.From)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		// FIXME: Create specialized error that formats it correctly to stderr
		if drop {
			printlnError("Failed to mod edit drop replacement of %q (command %q)", replacement.From, cmd)
		} else {
			printlnError("Failed to mod edit replace from %q to %q (command %q)", replacement.From, replacement.To, cmd)
		}

		os.Stderr.Write(output)
		os.Exit(1)
	}

	if bump {
		bumpDependency(ctx, input)
	}

	zlog.Debug("completed replacement", zap.Bool("drop", drop), zap.String("from", replacement.From), zap.String("to", replacement.From))
}

func bumpDependency(ctx context.Context, input string) {
	zlog.Debug("bumping dependency", zap.String("input", input))

	cmd := exec.CommandContext(ctx, "go_bump", input)

	output, err := cmd.CombinedOutput()
	if err != nil {
		printlnError("Failed to bump %q (command %q)", input, cmd)

		os.Stderr.Write(output)
		os.Exit(1)
	}
}

func printlnInfo(message string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, message+"\n", args...)
}

func printlnError(message string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, message+"\n", args...)
}

func getStringFlag(cmd *cobra.Command, key string) string {
	out, err := cmd.Flags().GetString(key)
	cli.NoError(err, key+" get string flag")

	return out
}

func getBoolFlag(cmd *cobra.Command, key string) bool {
	out, err := cmd.Flags().GetBool(key)
	cli.NoError(err, key+" get bool flag")

	return out
}
