package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/streamingfast/cli"
	. "github.com/streamingfast/cli"
	"go.uber.org/zap"
)

var HookInstall = Command(hookInstall,
	"install",
	"Install Git hooks to ensure that you do not push a local replacement",
	Flags(func(flags *pflag.FlagSet) {
		flags.BoolP("dry-run", "n", true, "Perform a dry-run and do not actually perform the install")
		flags.BoolP("force", "f", false, "Perform a real installation by writing hook(s) file into the repository")
		flags.BoolP("overwrite", "o", false, "Overwrites existing file if found")
		flags.String("skip", "", "Skip project matching flag's value (which is interpreted as a regex)")
	}),
)

func hookInstall(cmd *cobra.Command, args []string) error {
	root := "."
	if len(args) == 1 {
		root = args[0]
	}

	dryRun := getBoolFlag(cmd, "dry-run")
	overwrite := getBoolFlag(cmd, "overwrite")
	if getBoolFlag(cmd, "force") {
		dryRun = false
	}
	skip := getStringFlag(cmd, "skip")
	skipRegex := regexp.MustCompile(skip)

	var gitRepositories []string
	err := filepath.WalkDir(root, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			// Exclude some common know to be problematic folders
			if info.Name() == "node_modules" {
				return filepath.SkipDir
			}

			entries, err := os.ReadDir(path)
			if err != nil {
				printlnError("Unable to list directory %q entries: %w", path, err)
				return filepath.SkipDir
			}

			for _, entry := range entries {
				if entry.IsDir() && entry.Name() == ".git" {
					projectRoot := path
					zlog.Debug("found git repository", zap.String("git_dir", path), zap.String("project_root", projectRoot))

					if skip != "" && skipRegex.MatchString(info.Name()) {
						zlog.Debug("git repository is skipped according to skip pattern", zap.String("project_root", projectRoot), zap.String("skip", skip))
					} else {
						gitRepositories = append(gitRepositories, projectRoot)
					}

					return filepath.SkipDir
				}
			}
		}

		return nil
	})
	cli.NoError(err, "unable to complete walk of %q to find Git repositories", root)

	if dryRun {
		fmt.Println("Dry run mode, use '-f' (--force) to write hooks")
	}

	for _, gitRepository := range gitRepositories {
		hookIntoGitRepository(gitRepository, dryRun, overwrite)
	}

	return nil
}

func hookIntoGitRepository(path string, dryRun bool, overwrite bool) {
	zlog.Debug("hooking into Git repository, if contains some 'go.mod' files")
	directMatches, err := filepath.Glob(filepath.Join(path, "go.mod"))
	cli.NoError(err, "unable to check if git repository has some 'go.mod' files")

	subMatches, err := filepath.Glob(filepath.Join(path, "**/go.mod"))
	cli.NoError(err, "unable to check if git repository has some 'go.mod' files")

	if len(directMatches)+len(subMatches) == 0 {
		zlog.Debug("git repository does not seems like a Golang project (no go.mod files), skipping", zap.String("git_repository", path))
		return
	}

	gitDir := filepath.Join(path, ".git")
	hooksDir := filepath.Join(gitDir, "hooks")
	prePushHookFile := filepath.Join(hooksDir, "pre-push")
	alreadyExists := cli.FileExists(prePushHookFile)

	if dryRun {
		if alreadyExists && !overwrite {
			fmt.Printf("Would NOT overwrite existing pre-push hook at %s, use -o (--overwite) to overwrite it\n", prePushHookFile)
		} else if alreadyExists && overwrite {
			fmt.Printf("Would overwrite existing pre-push at %s\n", prePushHookFile)
		} else {
			fmt.Printf("Would write pre-push hook to %s\n", prePushHookFile)
		}

		return
	}

	if alreadyExists && !overwrite {
		printlnError("A pre-push hook already exists at %s but flag -o (--overwrite) was not passed to overwrite it", prePushHookFile)
		return
	}

	prePushContent := fmt.Sprintf(prePushContentTemplate, "v0.0.1", time.Now().Format(time.Kitchen))

	cli.NoError(os.MkdirAll(hooksDir, os.ModePerm), "unable to create %q directory", hooksDir)
	cli.NoError(os.WriteFile(prePushHookFile, []byte(prePushContent), os.ModePerm), "unable to write pre-push hook")
	fmt.Printf("Wrote pre-push hook to %s\n", prePushHookFile)
}

var prePushContentTemplate = `#!/bin/sh

# Created by go_replace %s (https://github.com/streamingfast/tooling#readme)
#   At: %s

exec go_replace hook verify "$1" "$2"
`
