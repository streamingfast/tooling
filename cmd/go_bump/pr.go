package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// upgradeEntry represents a single package upgrade extracted from `go get` output.
type upgradeEntry struct {
	// module is the full Go module path, e.g. "github.com/streamingfast/bstream"
	module string
	// fromVersion is the previous version string, e.g. "v0.1.0"
	fromVersion string
	// toVersion is the new version string, e.g. "v0.2.0"
	toVersion string
}

// upgradeLineRegex matches lines of the form:
//
//	go: upgraded <module> <from> => <to>
var upgradeLineRegex = regexp.MustCompile(`^go: upgraded (\S+) (\S+) => (\S+)$`)

// parseUpgradeEntries scans the raw output of `go get` and returns all upgrade
// entries it finds.
func parseUpgradeEntries(output string) []upgradeEntry {
	var entries []upgradeEntry
	for line := range strings.SplitSeq(output, "\n") {
		if m := upgradeLineRegex.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			entries = append(entries, upgradeEntry{
				module:      m[1],
				fromVersion: m[2],
				toVersion:   m[3],
			})
		}
	}
	return entries
}

// prBranchName returns the git branch name to use for the PR.  When there is
// exactly one upgraded module we use the last path component of the module
// path.  For multiple modules we fall back to a generic "bump-dependencies"
// name so the branch stays short and unambiguous.
func prBranchName(entries []upgradeEntry) string {
	if len(entries) == 1 {
		parts := strings.Split(entries[0].module, "/")
		return "bump/" + parts[len(parts)-1]
	}
	return "bump/dependencies"
}

// prCommitMessage builds the git commit message for the dependency bump.
func prCommitMessage(entries []upgradeEntry) string {
	switch len(entries) {
	case 0:
		return "chore: bump dependencies"
	case 1:
		e := entries[0]
		return fmt.Sprintf("chore: bump %s %s => %s", e.module, e.fromVersion, e.toVersion)
	default:
		var sb strings.Builder
		sb.WriteString("chore: bump dependencies\n\n")
		for _, e := range entries {
			fmt.Fprintf(&sb, "- %s %s => %s\n", e.module, e.fromVersion, e.toVersion)
		}
		return sb.String()
	}
}

// gitCurrentBranch returns the name of the current git branch.
func gitCurrentBranch(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// gitRemoteURL returns the URL of the "origin" remote, or an empty string when
// the remote is not configured.
func gitRemoteURL(ctx context.Context) string {
	out, err := exec.CommandContext(ctx, "git", "remote", "get-url", "origin").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// prURLFromRemote derives a GitHub "compare" URL from the remote URL so the
// user can open a PR manually when `gh` is not available.
//
// Supported remote formats:
//   - https://github.com/owner/repo.git
//   - git@github.com:owner/repo.git
func prURLFromRemote(remoteURL, baseBranch, headBranch string) string {
	repo := normalizeGitHubRepo(remoteURL)
	if repo == "" {
		return ""
	}
	return fmt.Sprintf("https://github.com/%s/compare/%s...%s", repo, baseBranch, headBranch)
}

// normalizeGitHubRepo extracts "owner/repo" from a GitHub remote URL, stripping
// any ".git" suffix.  It returns an empty string when the URL is not
// recognisable as a GitHub remote.
func normalizeGitHubRepo(remoteURL string) string {
	remoteURL = strings.TrimSuffix(remoteURL, ".git")

	// HTTPS form: https://github.com/owner/repo
	if rest, ok := strings.CutPrefix(remoteURL, "https://github.com/"); ok {
		return rest
	}

	// SSH form: git@github.com:owner/repo
	if rest, ok := strings.CutPrefix(remoteURL, "git@github.com:"); ok {
		return rest
	}

	return ""
}

// runPRMode performs the full PR-creation workflow around an already-resolved
// set of package IDs.  It creates a branch, bumps the dependencies, commits
// the changes, and either opens a PR with `gh` or prints a URL.
func runPRMode(ctx context.Context, packageIDs []PackageID, config *Config) error {
	// Remember the starting branch so we can return to it via defer.
	originalBranch, err := gitCurrentBranch(ctx)
	if err != nil {
		return fmt.Errorf("determine current branch: %w", err)
	}

	// Optimistically run `go get` first so we know which packages were actually
	// upgraded before we touch any git state.  This also means we leave the
	// working tree in the expected state without requiring a branch switch first.
	//
	// However, we need to create the branch *before* we modify go.mod/go.sum so
	// that the changes land on the right branch.  We therefore:
	//   1. Capture the current branch.
	//   2. Create + checkout the bump branch.
	//   3. Run bump().
	//   4. Stage, commit, push/open PR.
	//   5. Return to the original branch.

	// Pre-flight: make sure `go get` output contains at least one upgrade so we
	// don't create an empty PR.  We run bump() after the branch is created so
	// the go.mod changes land on the right branch.  Performing a "dry run" is
	// not straightforward, so we proceed and bail out before committing if
	// nothing changed.

	// Determine the bump branch name from the raw package IDs (before we know
	// exact upgrade versions) using a best-effort slug derived from the last
	// component of each resolved ID module path.
	bumpBranch := bumpBranchFromPackageIDs(packageIDs)

	// Create and check out the new branch.
	checkoutCmd := exec.CommandContext(ctx, "git", "checkout", "-b", bumpBranch)
	if out, err := checkoutCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("create branch %q: %w\n%s", bumpBranch, err, out)
	}

	// Ensure we return to the original branch when the function exits, regardless
	// of success or failure.
	defer func() {
		returnCmd := exec.CommandContext(ctx, "git", "checkout", originalBranch)
		if out, err := returnCmd.CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to return to branch %q: %v\n%s\n", originalBranch, err, out)
		}
	}()

	// Run the bump and capture the raw output so we can parse upgrade entries.
	goGetOutput, ok := bumpWithOutput(ctx, packageIDs...)
	if !ok {
		// bumpWithOutput already printed the error; just return a sentinel.
		return fmt.Errorf("go get failed")
	}

	upgrades := parseUpgradeEntries(goGetOutput)
	if len(upgrades) == 0 {
		// Nothing was actually upgraded.  Switch back and report.
		return fmt.Errorf("no packages were upgraded; nothing to commit")
	}

	// Optionally run go mod tidy if configured.
	if config.AfterBump.GoModTidy {
		runGoModTidy(ctx)
	}

	// Stage go.mod and go.sum.
	addCmd := exec.CommandContext(ctx, "git", "add", "go.mod", "go.sum")
	if out, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add: %w\n%s", err, out)
	}

	// Create the commit.
	commitMsg := prCommitMessage(upgrades)
	commitCmd := exec.CommandContext(ctx, "git", "commit", "-m", commitMsg)
	if out, err := commitCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit: %w\n%s", err, out)
	}

	// Open or print the PR.
	if _, err := exec.LookPath("gh"); err == nil {
		return openPRWithGH(ctx, originalBranch, commitMsg)
	}

	return pushAndPrintPRURL(ctx, originalBranch, bumpBranch)
}

// bumpBranchFromPackageIDs derives a branch name from the resolved package IDs
// before we know exact upgraded versions.
func bumpBranchFromPackageIDs(packageIDs []PackageID) string {
	if len(packageIDs) == 1 {
		// Strip any @version suffix, then take the last path component.
		module, _, _ := strings.Cut(string(packageIDs[0]), "@")
		parts := strings.Split(module, "/")
		return "bump/" + parts[len(parts)-1]
	}
	return "bump/dependencies"
}

// bumpWithOutput is like bump() but returns the raw combined output together
// with a boolean indicating success.
func bumpWithOutput(ctx context.Context, packageIDs ...PackageID) (output string, ok bool) {
	args := make([]string, 1+len(packageIDs))
	args[0] = "get"
	for i, id := range packageIDs {
		args[i+1] = string(id)
	}

	cmd := exec.CommandContext(ctx, "go", args...)
	rawOutput, err := cmd.CombinedOutput()
	output = string(rawOutput)
	fmt.Print(output)

	if err != nil {
		printlnError("Failed to bump packages %s (command %q)", strings.Join(args[1:], ", "), cmd)
		return output, false
	}
	return output, true
}

// openPRWithGH uses the `gh` CLI to create a pull request.
func openPRWithGH(ctx context.Context, baseBranch, commitMsg string) error {
	// Use the first line of the commit message as the PR title.
	title, _, _ := strings.Cut(commitMsg, "\n")

	ghCmd := exec.CommandContext(ctx, "gh", "pr", "create",
		"--base", baseBranch,
		"--title", title,
		"--body", commitMsg,
	)
	ghCmd.Stdout = os.Stdout
	ghCmd.Stderr = os.Stderr
	if err := ghCmd.Run(); err != nil {
		return fmt.Errorf("gh pr create: %w", err)
	}
	return nil
}

// pushAndPrintPRURL pushes the current branch and prints the GitHub compare URL.
func pushAndPrintPRURL(ctx context.Context, baseBranch, headBranch string) error {
	pushCmd := exec.CommandContext(ctx, "git", "push", "--set-upstream", "origin", headBranch)
	pushCmd.Stdout = os.Stdout
	pushCmd.Stderr = os.Stderr
	if err := pushCmd.Run(); err != nil {
		return fmt.Errorf("git push: %w", err)
	}

	remoteURL := gitRemoteURL(ctx)
	prURL := prURLFromRemote(remoteURL, baseBranch, headBranch)
	if prURL != "" {
		fmt.Printf("\nOpen a pull request at:\n  %s\n", prURL)
	} else {
		fmt.Printf("\nBranch %q pushed. Open a PR targeting %q manually.\n", headBranch, baseBranch)
	}
	return nil
}
