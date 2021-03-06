package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/dfuse-io/logging"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var rootCmd = &cobra.Command{
	Use:          "go_bump <id> [<id>...]",
	Short:        "Bumps dependency id(s) provided on the currently dectected Golang module, commit the changes and create a commit with a standardize message",
	Long:         usage(),
	RunE:         run,
	Args:         cobra.MinimumNArgs(1),
	SilenceUsage: true,
}

var zlog = logging.NewSimpleLogger("go_dump", "github.com/dfuse-io/tooling/cmd/go_bump")

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
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

	for _, arg := range args {
		resolved, err := UnresolvedPackageID(arg).Resolve(config)
		if err != nil {
			return fmt.Errorf("package id resolution: %w", err)
		}

		bump(cmd.Context(), resolved)
	}

	return nil
}

func bump(ctx context.Context, packageID PackageID) {
	cmd := exec.CommandContext(ctx, "go", "get", string(packageID))
	output, err := cmd.CombinedOutput()
	if err != nil {
		// FIXME: Create specialized error that formats it correctly to stderr
		printlnError("Failed to bump package %s (command %q)", packageID, cmd)
		os.Stderr.Write(output)
		os.Exit(1)
	}

	zlog.Debug("completed bumping of package", zap.String("package_id", string(packageID)))
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

  GitHub Shortcut               @dfuse-io/bstream@develop (equivalent to github.com/dfuse-io/bstream@develop)
  Default Project Shortcut      ~bstream@develop (equivalent to github.com/dfuse-io/bstream@develop)
  Default Branch Shortcut       ~bstream! (equivalent to github.com/dfuse-io/bstream@develop)

You can configure the default values used for ~ and ! via config file '$HOME/config/go_bump/default.yaml'
that if exists, can override the values

  default_repo_shortcut: github.com                # To define value of leading @
  default_project_shortcut: github.com/something   # To define value of leading ~
  default_branch_shortcut: @develop                # To define value of trailing !

If it does *not* start with a ~ or @ character, it is considered a 'go get' format.
The '!' is still expanded in all situation if it terminates the id.
`
}
