package main

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	. "github.com/streamingfast/cli"
	"github.com/streamingfast/cli/sflags"
)

var ComposeGroup = Group(
	"compose",
	"Manage Docker Compose projects",
	Command(composeDownAllE, "down-all",
		"Bring down every running Docker Compose project",
		Flags(func(flags *pflag.FlagSet) {
			flags.StringSlice("except", nil, "Project name(s) to leave running, repeat the flag or separate them with a comma")
			flags.Bool("keep-volumes", false, "Keep the projects volumes instead of deleting them")
			flags.BoolP("yes", "y", false, "Do not ask for confirmation before bringing the projects down")
		}),
		Description(`
			Lists every running Docker Compose project (like 'docker compose ls' does) and
			runs 'docker compose down' on each of them, deleting their volumes along the way.

			Volumes are deleted by default, use '--keep-volumes' to keep them.

			The list of projects that are about to be brought down is printed and confirmation
			is asked before anything is deleted, use '--yes' to skip the confirmation. When
			the command is not attached to a terminal (in a script for example), it aborts
			unless '--yes' is provided.

			A project that fails to come down does not prevent the other ones from being
			processed, all the failures are reported once every project has been handled.
		`),
		Example(`
			# Bring down every running project, deleting their volumes
			dockerx compose down-all

			# Same but keep the volumes around
			dockerx compose down-all --keep-volumes

			# Leave the 'my-project' project running
			dockerx compose down-all --except my-project

			# Leave multiple projects running
			dockerx compose down-all --except my-project,other-project

			# Do not ask for confirmation
			dockerx compose down-all --yes
		`),
	),
)

// composeProject is a single entry of the 'docker compose ls --format json' output.
type composeProject struct {
	Name        string `json:"Name"`
	Status      string `json:"Status"`
	ConfigFiles string `json:"ConfigFiles"`
}

func composeDownAllE(cmd *cobra.Command, args []string) error {
	except := sflags.MustGetStringSlice(cmd, "except")
	keepVolumes := sflags.MustGetBool(cmd, "keep-volumes")
	assumeYes := sflags.MustGetBool(cmd, "yes")

	output, err := docker("compose", "ls", "--format", "json").Output()
	if err != nil {
		return fmt.Errorf("listing compose projects: %w", err)
	}

	projects, err := parseComposeProjects(output)
	if err != nil {
		return err
	}

	if len(projects) == 0 {
		fmt.Println("No running Docker Compose project found")
		return nil
	}

	toBringDown, skipped := partitionProjects(projects, except)

	for _, project := range skipped {
		fmt.Printf("Skipping %s (%s)\n", project.Name, project.Status)
	}

	if len(toBringDown) == 0 {
		fmt.Println("No running Docker Compose project left to bring down")
		return nil
	}

	volumesNotice := "deleting their volumes"
	if keepVolumes {
		volumesNotice = "keeping their volumes"
	}

	fmt.Printf("About to bring down %d project(s), %s:\n", len(toBringDown), volumesNotice)
	for _, project := range toBringDown {
		fmt.Printf("  - %s (%s)\n", project.Name, project.Status)
	}
	fmt.Println()

	if !assumeYes {
		confirmed, wasAnswered := AskConfirmation("Bring down those %d project(s)?", len(toBringDown))
		if !wasAnswered || !confirmed {
			return fmt.Errorf("aborted")
		}
	}

	var failures []string
	for _, project := range toBringDown {
		fmt.Printf("Bringing down %s\n", project.Name)

		if err := docker(composeDownArgs(project.Name, keepVolumes)...).Run(); err != nil {
			fmt.Printf("Failed to bring down %s: %s\n", project.Name, err)
			failures = append(failures, project.Name)
		}
	}

	if len(failures) != 0 {
		return fmt.Errorf("unable to bring down %d project(s): %s", len(failures), strings.Join(failures, ", "))
	}

	fmt.Printf("Brought down %d project(s)\n", len(toBringDown))
	return nil
}

// parseComposeProjects decodes the 'docker compose ls --format json' output, discarding
// unnamed entries and sorting the remaining ones by name so the output is deterministic.
func parseComposeProjects(output []byte) ([]composeProject, error) {
	if len(strings.TrimSpace(string(output))) == 0 {
		return nil, nil
	}

	var decoded []composeProject
	if err := json.Unmarshal(output, &decoded); err != nil {
		return nil, fmt.Errorf("decoding 'docker compose ls' output: %w", err)
	}

	var projects []composeProject
	for _, project := range decoded {
		if project.Name != "" {
			projects = append(projects, project)
		}
	}

	slices.SortFunc(projects, func(a, b composeProject) int { return strings.Compare(a.Name, b.Name) })

	return projects, nil
}

// partitionProjects splits projects between the ones to bring down and the ones excluded
// by the 'except' project names, an exact (case sensitive) match on the project's name.
func partitionProjects(projects []composeProject, except []string) (toBringDown, skipped []composeProject) {
	excluded := make(map[string]bool, len(except))
	for _, name := range except {
		if name = strings.TrimSpace(name); name != "" {
			excluded[name] = true
		}
	}

	for _, project := range projects {
		if excluded[project.Name] {
			skipped = append(skipped, project)
		} else {
			toBringDown = append(toBringDown, project)
		}
	}

	return toBringDown, skipped
}

// composeDownArgs builds the 'docker' arguments bringing a single project down.
func composeDownArgs(project string, keepVolumes bool) []string {
	args := []string{"compose", "--project-name", project, "down"}
	if !keepVolumes {
		args = append(args, "--volumes")
	}

	return args
}
