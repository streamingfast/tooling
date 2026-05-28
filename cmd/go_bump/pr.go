package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	. "github.com/streamingfast/cli"
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

// moduleShortName returns the last path component of a module path, stripping
// any @version suffix first.
func moduleShortName(module string) string {
	base, _, _ := strings.Cut(module, "@")
	parts := strings.Split(base, "/")
	return parts[len(parts)-1]
}

// modulePath returns the bare module path of a (possibly versioned) package ID,
// stripping any "@version" suffix, e.g. "github.com/streamingfast/eth-go@develop"
// becomes "github.com/streamingfast/eth-go".
func modulePath(id string) string {
	base, _, _ := strings.Cut(id, "@")
	return base
}

// unupgradedRequestedPackages returns the bare module paths of the requested
// package IDs that were NOT actually upgraded by `go get`. An empty result
// means every requested package was upgraded.
func unupgradedRequestedPackages(packageIDs []PackageID, upgrades []upgradeEntry) []string {
	upgraded := make(map[string]struct{}, len(upgrades))
	for _, u := range upgrades {
		upgraded[u.module] = struct{}{}
	}

	var missing []string
	for _, id := range packageIDs {
		path := modulePath(string(id))
		if _, ok := upgraded[path]; !ok {
			missing = append(missing, path)
		}
	}
	return missing
}

// preBumpBranchName returns the initial branch name to use before `go get`
// runs.  For a single package we use the short name only; for 2–3 packages we
// join their short names; for 4+ we use "bump/dependencies".
func preBumpBranchName(packageIDs []PackageID) string {
	switch len(packageIDs) {
	case 1:
		return "bump/" + moduleShortName(string(packageIDs[0]))
	case 2, 3:
		parts := make([]string, len(packageIDs))
		for i, id := range packageIDs {
			parts[i] = moduleShortName(string(id))
		}
		return "bump/" + strings.Join(parts, "-")
	default:
		return "bump/dependencies"
	}
}

// finalBranchName computes the definitive branch name after the bump.
// For a single upgraded module the name includes the new version; for
// multiple modules (or when there are no upgrades) the pre-bump name is
// returned unchanged.
func finalBranchName(preBranch string, upgrades []upgradeEntry, packageIDs []PackageID) string {
	if len(packageIDs) == 1 && len(upgrades) > 0 {
		short := moduleShortName(upgrades[0].module)
		return fmt.Sprintf("bump/%s-to-%s", short, upgrades[0].toVersion)
	}
	return preBranch
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

// gitBranchExists reports whether a local branch with the given name exists.
func gitBranchExists(ctx context.Context, branch string) bool {
	err := exec.CommandContext(ctx, "git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch).Run()
	return err == nil
}

// revertGoModChanges restores go.mod and go.sum to their committed state,
// discarding any modifications made by `go get` during the aborted bump. This
// is used when the requested package was not upgraded so the repository is left
// exactly as it was before the command ran.
func revertGoModChanges(ctx context.Context) {
	cmd := exec.CommandContext(ctx, "git", "checkout", "--", "go.mod", "go.sum")
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to revert go.mod/go.sum changes: %v\n%s\n", err, out)
		return
	}
	fmt.Fprintln(os.Stderr, "Reverted go.mod/go.sum changes.")
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
	originalBranch, err := gitCurrentBranch(ctx)
	if err != nil {
		return fmt.Errorf("determine current branch: %w", err)
	}

	// Compute the initial branch name before we know the exact new version.
	// For a single package this is "bump/<short>"; we rename it after `go get`.
	initialBranch := preBumpBranchName(packageIDs)

	if gitBranchExists(ctx, initialBranch) {
		confirmed, wasAnswered := AskConfirmation("Branch %q already exists. Delete it and continue?", initialBranch)
		if !wasAnswered || !confirmed {
			return fmt.Errorf("aborted: branch %q already exists", initialBranch)
		}
		if out, err := exec.CommandContext(ctx, "git", "branch", "-D", initialBranch).CombinedOutput(); err != nil {
			return fmt.Errorf("delete branch %q: %w\n%s", initialBranch, err, out)
		}
	}

	// Create and check out the initial bump branch.
	if out, err := exec.CommandContext(ctx, "git", "checkout", "-b", initialBranch).CombinedOutput(); err != nil {
		return fmt.Errorf("create branch %q: %w\n%s", initialBranch, err, out)
	}

	// Ensure we return to the original branch when the function exits.
	defer func() {
		returnCmd := exec.CommandContext(ctx, "git", "checkout", originalBranch)
		if out, err := returnCmd.CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to return to branch %q: %v\n%s\n", originalBranch, err, out)
		}
	}()

	// Run `go get` — output is printed as it goes via goGet writing to os.Stdout.
	goGetOutput, ok := goGet(ctx, os.Stdout, packageIDs...)
	if !ok {
		return fmt.Errorf("go get failed")
	}

	upgrades := parseUpgradeEntries(goGetOutput)
	if len(upgrades) == 0 {
		return fmt.Errorf("no packages were upgraded; nothing to commit")
	}

	// Validate that the packages the user actually asked for were upgraded.
	// `go get` can upgrade transitive dependencies even when a requested
	// package was already at the desired version; creating a PR for those
	// unrelated bumps is almost never what the user wanted.
	if missing := unupgradedRequestedPackages(packageIDs, upgrades); len(missing) > 0 {
		printlnError("The following requested package(s) were NOT upgraded (they were likely already at the requested version):")
		for _, m := range missing {
			printlnError("  - %s", m)
		}
		if len(upgrades) > 0 {
			printlnError("However, `go get` did upgrade the following (transitive) dependencies:")
			for _, u := range upgrades {
				printlnError("  - %s %s => %s", u.module, u.fromVersion, u.toVersion)
			}
		}

		confirmed, wasAnswered := AskConfirmation("None of the requested packages were bumped. Revert these unrelated changes?")
		if wasAnswered && confirmed {
			revertGoModChanges(ctx)
		}
		return fmt.Errorf("aborted: requested package(s) not upgraded: %s", strings.Join(missing, ", "))
	}

	// For a single package, rename the branch to include the new version.
	bumpBranch := finalBranchName(initialBranch, upgrades, packageIDs)
	if bumpBranch != initialBranch {
		if gitBranchExists(ctx, bumpBranch) {
			confirmed, wasAnswered := AskConfirmation("Branch %q already exists. Delete it and continue?", bumpBranch)
			if !wasAnswered || !confirmed {
				return fmt.Errorf("aborted: branch %q already exists", bumpBranch)
			}
			if out, err := exec.CommandContext(ctx, "git", "branch", "-D", bumpBranch).CombinedOutput(); err != nil {
				return fmt.Errorf("delete branch %q: %w\n%s", bumpBranch, err, out)
			}
		}
		if out, err := exec.CommandContext(ctx, "git", "branch", "-m", initialBranch, bumpBranch).CombinedOutput(); err != nil {
			return fmt.Errorf("rename branch to %q: %w\n%s", bumpBranch, err, out)
		}
	}

	if config.AfterBump.GoModTidy {
		runGoModTidy(ctx)
	}

	// Stage go.mod and go.sum.
	if out, err := exec.CommandContext(ctx, "git", "add", "go.mod", "go.sum").CombinedOutput(); err != nil {
		return fmt.Errorf("git add: %w\n%s", err, out)
	}

	// Create the commit.
	commitMsg := prCommitMessage(upgrades)
	if out, err := exec.CommandContext(ctx, "git", "commit", "-m", commitMsg).CombinedOutput(); err != nil {
		return fmt.Errorf("git commit: %w\n%s", err, out)
	}

	// Push the branch to the remote so that `gh pr create` (or a manual URL)
	// can reference it.
	pushCmd := exec.CommandContext(ctx, "git", "push", "--set-upstream", "origin", bumpBranch)
	pushCmd.Stdout = os.Stdout
	pushCmd.Stderr = os.Stderr
	if err := pushCmd.Run(); err != nil {
		return fmt.Errorf("git push: %w", err)
	}

	if _, err := exec.LookPath("gh"); err == nil {
		return openPRWithGH(ctx, originalBranch, bumpBranch, commitMsg)
	}

	return printPRURL(ctx, originalBranch, bumpBranch)
}

// openPRWithGH uses the `gh` CLI to create a pull request.
func openPRWithGH(ctx context.Context, baseBranch, headBranch, commitMsg string) error {
	title, _, _ := strings.Cut(commitMsg, "\n")

	ghCmd := exec.CommandContext(ctx, "gh", "pr", "create",
		"--base", baseBranch,
		"--head", headBranch,
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

// printPRURL prints a GitHub compare URL the user can open manually.
func printPRURL(ctx context.Context, baseBranch, headBranch string) error {
	remoteURL := gitRemoteURL(ctx)
	prURL := prURLFromRemote(remoteURL, baseBranch, headBranch)
	if prURL != "" {
		fmt.Printf("\nOpen a pull request at:\n  %s\n", prURL)
	} else {
		fmt.Printf("\nBranch %q pushed. Open a PR targeting %q manually.\n", headBranch, baseBranch)
	}
	return nil
}
