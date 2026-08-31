package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

// CurrentContext represents using the current kubectl context namespace (empty string).
const CurrentContext = ""

// kubectl creates a new kubectl command builder with the given namespace and arguments.
// Use CurrentContext (or "") to use the current kubectl context namespace.
func kubectl(ns string, args ...string) *kubectlCmd {
	return &kubectlCmd{
		namespace: ns,
		args:      args,
		stdout:    os.Stdout,
		stderr:    os.Stderr,
	}
}

type kubectlCmd struct {
	namespace string
	args      []string
	stdin     io.Reader
	stdout    io.Writer
	stderr    io.Writer
}

// Namespace overrides the namespace for the kubectl command.
func (k *kubectlCmd) Namespace(ns string) *kubectlCmd {
	k.namespace = ns
	return k
}

// Stdin sets the stdin reader for the kubectl command.
func (k *kubectlCmd) Stdin(r io.Reader) *kubectlCmd {
	k.stdin = r
	return k
}

// Stdout sets the stdout writer for the kubectl command.
func (k *kubectlCmd) Stdout(w io.Writer) *kubectlCmd {
	k.stdout = w
	return k
}

// Stderr sets the stderr writer for the kubectl command.
func (k *kubectlCmd) Stderr(w io.Writer) *kubectlCmd {
	k.stderr = w
	return k
}

// Silent disables stdout and stderr output.
func (k *kubectlCmd) Silent() *kubectlCmd {
	k.stdout = nil
	k.stderr = nil
	return k
}

// buildArgs constructs the final argument list with namespace prepended if set.
func (k *kubectlCmd) buildArgs() []string {
	if k.namespace != "" {
		return append([]string{"-n", k.namespace}, k.args...)
	}
	return k.args
}

// Run executes the kubectl command and returns an error if it fails.
func (k *kubectlCmd) Run() error {
	cmd := exec.Command("kubectl", k.buildArgs()...)
	cmd.Stdin = k.stdin
	cmd.Stdout = k.stdout
	cmd.Stderr = k.stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("kubectl %v failed: %w", k.args, err)
	}
	return nil
}

// Output executes the kubectl command and returns stdout as bytes.
func (k *kubectlCmd) Output() ([]byte, error) {
	cmd := exec.Command("kubectl", k.buildArgs()...)
	cmd.Stdin = k.stdin
	cmd.Stderr = k.stderr

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("kubectl %v failed: %w", k.args, err)
	}
	return output, nil
}

// Success executes the kubectl command and returns true if it succeeds.
func (k *kubectlCmd) Success() bool {
	cmd := exec.Command("kubectl", k.buildArgs()...)
	cmd.Stdin = k.stdin
	return cmd.Run() == nil
}
