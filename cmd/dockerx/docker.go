package main

import (
	"fmt"
	"os"
	"os/exec"
)

// docker creates a new docker command builder with the given arguments.
func docker(args ...string) *dockerCmd {
	return &dockerCmd{
		args:   args,
		stdout: os.Stdout,
		stderr: os.Stderr,
	}
}

type dockerCmd struct {
	args   []string
	stdout *os.File
	stderr *os.File
}

// Run executes the docker command, streaming its output, and returns an error if it fails.
func (d *dockerCmd) Run() error {
	cmd := exec.Command("docker", d.args...)
	cmd.Stdout = d.stdout
	cmd.Stderr = d.stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker %v failed: %w", d.args, err)
	}
	return nil
}

// Output executes the docker command and returns stdout as bytes.
func (d *dockerCmd) Output() ([]byte, error) {
	cmd := exec.Command("docker", d.args...)
	cmd.Stderr = d.stderr

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker %v failed: %w", d.args, err)
	}
	return output, nil
}
