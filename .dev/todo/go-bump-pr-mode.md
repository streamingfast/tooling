# Add --pr Mode to go_bump

mode: feature
state: ready
root_git: .
worktree: .worktrees/feature/go-bump-pr-mode
branch: feature/go-bump-pr-mode
target_branch: master

> **Resume protocol:** read **Dev Feedback** and the **State Tracker** below first, then jump to the
> step marked `Current`. Ensure that you are in the correct worktree and branch according to preamble here. Update current with Developer feedback and update the tracker after every meaningful change.
> Do not mutate completed steps; append a new entry instead.

---

## Initial Description

Add a `--pr` mode to `go_bump` that would:
- Checkout a branch preparing a PR for the bump
    - Keep current branch to be able to return to it once ready
    - Ensure this is done in some defer step to ensure we return to normal branch
- Bump like we do right now
- Git add go.mod/go.sum file
- Generate a correct commit with message + details about the bump (from_version => to_version)
- Using `gh` if available, open a PR on GitHub on behalf of the user
    - Use kept "current branch" as target
- If `gh` is not available, git push and present user's the URL to open a PR for the branch (based on git remote)
    - Use kept "current branch" as target

## Dev Feedback

## Spec & Implementation

## State Tracker

**Last Updated:** 2026-05-20
**Current Step:** Step 0 — Not started
**Status:** Ready for implementation
