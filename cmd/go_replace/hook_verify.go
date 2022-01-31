package main

import (
	"bufio"
	"context"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/streamingfast/cli"
	. "github.com/streamingfast/cli"
	"go.uber.org/zap"
	"golang.org/x/mod/modfile"
)

var HookVerify = Command(hookVerify,
	"verify",
	"Verify hook received data to ensure that you do not push a local replacement (normally called by Git directly)",
)

var zeroHash = "0000000000000000000000000000000000000000"

func hookVerify(cmd *cobra.Command, args []string) error {
	remoteName := args[0]
	remoteURL := args[1]
	zlog.Debug("hook verify invoked", zap.String("remote_name", remoteName), zap.String("remote_ur", remoteURL))
	if os.Getenv("SKIP_HOOKS") == "true" {
		zlog.Debug("environment variable SKIP_HOOKS is defined, skipping hooks verification")
		return nil
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, " ")
		zlog.Debug("received commit info line", zap.Strings("parts", parts), zap.String("line", line))
		cli.Ensure(len(parts) == 4, "Expected stdin line of the form <local_ref> <local_oid> <remote_ref> <remote_oid>")

		localRef := parts[0]
		localObjectID := parts[1]
		remoteRef := parts[2]
		remoteObjectID := parts[3]

		_ = localRef
		_ = remoteRef

		// We only check if `localObjectID` is not the zero hash, otherwise it's a delete commit and we don't care
		if localObjectID != zeroHash {
			commitRange := ""
			if remoteObjectID == zeroHash {
				commitRange = localObjectID
			} else {
				commitRange = remoteObjectID + ".." + localObjectID
			}

			commits := fetchCommitsModifyingGoMod(cmd.Context(), commitRange)
			if len(commits) == 0 {
				zlog.Debug("no commits is modifying go.mod")
				return nil
			}

			foundLocalReplacement := checkAllGoModForLocalReplacement()
			if foundLocalReplacement {
				os.Exit(1)
			}

			zlog.Debug("found no replacement, push can continue normally")
		}
	}

	return nil
}

func checkAllGoModForLocalReplacement() (foundLocalReplacement bool) {
	workingDir, err := os.Getwd()
	cli.NoError(err, "unable to get working directory")

	// FIXME: Find local .git directory
	zlog.Debug("looking in working directory if any go.mod has local replacement", zap.String("working_dir", workingDir))
	err = filepath.WalkDir(workingDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// FIXME: Check if path is an excluded git directory
		if d.IsDir() {
			return nil
		}

		if d.Name() == "go.mod" {
			found := checkGoModForLocalReplacement(path)
			if found {
				foundLocalReplacement = true
			}
		}

		return nil
	})
	cli.NoError(err, "unable to walk working directory %q", workingDir)

	return
}

var windowsAbsolutePathDiskRegex = regexp.MustCompile(`^[a-zA-Z]:\\`)

func checkGoModForLocalReplacement(path string) (foundLocalReplacement bool) {
	zlog.Debug("parsing go.mod file and looking for local replacement", zap.String("path", path))

	content, err := os.ReadFile(path)
	cli.NoError(err, "unable to read go.mod file")

	module, err := modfile.Parse(path, content, nil)
	if err != nil {
		printlnError("Unable to read go.mod file at %q: %q", path, err)
		return
	}

	zlog.Debug("parsed module, inspecting replace directives", zap.Int("replace_count", len(module.Replace)))
	for _, replace := range module.Replace {
		new := replace.New.Path

		// Add other common repository prefixes, if the new path starts with that, it's definitely not a local path
		if strings.HasPrefix(new, "github.com/") || strings.HasPrefix(new, "gitlab.com/") {
			continue
		}

		// If it starts with '../', './', '/' or '[a-zA-Z]:\', it's definitely a local path
		if strings.HasPrefix(new, "../") || strings.HasPrefix(new, "./") || strings.HasPrefix(new, "/") || windowsAbsolutePathDiskRegex.MatchString(new) {
			reportLocalReplacementFound(path, new)
			foundLocalReplacement = true
			continue
		}

		zlog.Debug("replacement does not match any known rule, checking if it's not a directory")
		if cli.DirectoryExists(new) {
			reportLocalReplacementFound(path, new)
			foundLocalReplacement = true
			continue
		}

		zlog.Debug("replacement exhausted all checks, assuming it's non local path replacement")
	}

	return foundLocalReplacement
}

// fetchCommitsModifyingGoMod retrieves the list of commits that touched a `go.mod` file
// anywhere in the project.
//
// Note: There can be false positive because the glob pattern we use is "*go.mod" so it
// could match `somego.mod` for example.
func fetchCommitsModifyingGoMod(ctx context.Context, commitRange string) []string {
	cmd := exec.CommandContext(ctx, "git", "rev-list", commitRange, "--", "*go.mod")
	zlog.Debug("perform git rev-list to obtain commits modifying go.mod", zap.Stringer("cmd", cmd))

	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.HasPrefix(string(output), "fatal: Invalid revision range "+commitRange) {
			zlog.Debug("Seems like remote commit is not known locally, likely due to not being up-to-date with remote, letting it continue")
			os.Exit(0)
		}

		printlnError("Command %s failed", cmd.String())
		os.Stderr.Write(output)
		os.Exit(1)
	}

	var lines []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}

	zlog.Debug("retrieved list of commits that modified go.mod", zap.Strings("lines", lines))
	return lines
}

func reportLocalReplacementFound(moduleFilePath string, foundElement string) {
	workingDirectory, err := os.Getwd()
	cli.NoError(err, "unable to get working directory")

	relativePath, err := filepath.Rel(workingDirectory, moduleFilePath)
	cli.NoError(err, "unable to make module path %q relative to %q", moduleFilePath, workingDirectory)

	printlnError("The %s file has a local replacement %q, remove it prior pushing to remote repository", relativePath, foundElement)
}
