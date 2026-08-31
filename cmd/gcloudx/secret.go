package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	. "github.com/streamingfast/cli"
	"github.com/streamingfast/cli/sflags"
)

var SecretGroup = Group(
	"secret",
	"Manage Kubernetes secrets",
	Execute(secretListE),
	Command(secretShowE, "show <secret_name>",
		"Show all key-value pairs in a Kubernetes secret (decoded)",
		Description(`
			Displays all key-value pairs in a Kubernetes secret with values decoded from base64.
		`),
		Example(`
			# Show all keys and decoded values
			k8x secret show my-secret

			# Show secret from a specific namespace
			k8x -n production secret show my-secret
		`),
	),
	Command(secretSetE, "set <secret_name> <key>=<value> [<key>=<value> ...]",
		"Set one or more key-value pairs in a Kubernetes secret",
		Description(`
			Sets or updates key-value pairs in a Kubernetes secret.
			Creates the secret if it doesn't exist, or patches it if it does.

			If arguments are missing, interactive prompts will guide you through the process.
		`),
		Example(`
			# Set a single key with explicit value
			k8x secret set my-secret api_key=my-secret-value

			# Set multiple keys at once
			k8x secret set my-secret key1=value1 key2=value2

			# Use a specific namespace
			k8x -n production secret set my-secret api_key=value

			# Value from clipboard (pipe it yourself)
			pbpaste | xargs -I {} k8x secret set my-secret api_key={}
		`),
	),
	Command(secretTransferE, "transfer <source_namespace> <target_namespace> [<secret_name> ...]",
		"Transfer secrets from one namespace to another",
		Description(`
			Copies secrets from a source namespace to a target namespace, stripping
			cluster-specific metadata (uid, resourceVersion, managedFields, etc.)
			and updating the namespace reference.

			If secret names are provided as arguments they are transferred directly.
			Otherwise lists all secrets in the source namespace and prompts for
			selection: enter comma-separated names or "all" to transfer every secret.

			Existing secrets in the target namespace are fully replaced.
		`),
		Example(`
			# Transfer specific secrets directly (no prompt)
			k8x secret transfer staging production my-secret other-secret

			# Transfer secrets interactively (prompts for selection)
			k8x secret transfer staging production
			> Secret(s) to transfer: my-secret,other-secret

			# Transfer all secrets interactively
			k8x secret transfer staging production
			> Secret(s) to transfer: all

			# Prompts are shown if namespaces are omitted
			k8x secret transfer
		`),
	),
)

func secretListE(cmd *cobra.Command, args []string) error {
	ns := sflags.MustGetString(cmd, "namespace")

	return kubectl(ns, "get", "secret").Run()
}

func secretShowE(cmd *cobra.Command, args []string) error {
	ns := sflags.MustGetString(cmd, "namespace")

	var secretName string
	if len(args) >= 1 {
		secretName = args[0]
	}

	if secretName == "" {
		secretName = Prompt("Enter the secret name", PromptTypeString)
	}

	output, err := kubectl(ns, "get", "secret", secretName, "-o", "jsonpath={.data}").Output()
	if err != nil {
		return err
	}

	var data map[string]string
	if err := json.Unmarshal(output, &data); err != nil {
		return fmt.Errorf("parsing secret data: %w", err)
	}

	if len(data) == 0 {
		fmt.Printf("Secret %q has no data\n", secretName)
		return nil
	}

	for key, encodedValue := range data {
		decoded, err := base64.StdEncoding.DecodeString(encodedValue)
		if err != nil {
			fmt.Printf("%s: <decode error: %v>\n", key, err)
			continue
		}
		fmt.Printf("%s: %s\n", key, string(decoded))
	}

	return nil
}

func secretSetE(cmd *cobra.Command, args []string) error {
	ns := sflags.MustGetString(cmd, "namespace")

	var secretName string
	var keyValues []string

	if len(args) >= 1 {
		secretName = args[0]
		keyValues = args[1:]
	}

	if secretName == "" {
		secretName = Prompt("Enter the secret name", PromptTypeString)
	}

	if len(keyValues) == 0 {
		for {
			kv := Prompt("Enter key=value (empty to finish)", PromptTypeString)
			if kv == "" {
				break
			}
			keyValues = append(keyValues, kv)
		}
	}

	if len(keyValues) == 0 {
		return fmt.Errorf("at least one key=value pair is required")
	}

	data := make(map[string]string)
	for _, kv := range keyValues {
		key, value, err := parseKeyValue(kv)
		if err != nil {
			return fmt.Errorf("parsing key-value %q: %w", kv, err)
		}

		data[key] = base64.StdEncoding.EncodeToString([]byte(value))
	}

	exists := kubectl(ns, "get", "secret", secretName, "-o", "name").Silent().Success()

	if exists {
		return patchSecret(ns, secretName, data)
	}
	return createSecret(ns, secretName, data)
}

func createSecret(ns, name string, data map[string]string) error {
	manifest := map[string]any{
		"apiVersion": "v1",
		"kind":       "Secret",
		"metadata":   map[string]any{"name": name},
		"type":       "Opaque",
		"data":       data,
	}

	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("marshaling secret manifest: %w", err)
	}

	fmt.Printf("Creating secret %q with %d key(s)\n", name, len(data))

	return kubectl(ns, "create", "-f", "-").Stdin(strings.NewReader(string(manifestJSON))).Run()
}

func patchSecret(ns, name string, data map[string]string) error {
	patchJSON, err := json.Marshal(map[string]any{"data": data})
	if err != nil {
		return fmt.Errorf("marshaling patch: %w", err)
	}

	fmt.Printf("Patching secret %q with %d key(s)\n", name, len(data))

	return kubectl(ns, "patch", "secret", name, "-p", string(patchJSON)).Run()
}

func secretTransferE(cmd *cobra.Command, args []string) error {
	var sourceNS, targetNS string

	if len(args) >= 1 {
		sourceNS = args[0]
	}
	if len(args) >= 2 {
		targetNS = args[1]
	}

	if sourceNS == "" {
		sourceNS = Prompt("Enter the source namespace", PromptTypeString)
	}
	if targetNS == "" {
		targetNS = Prompt("Enter the target namespace", PromptTypeString)
	}

	// Secret names provided directly as arguments — skip listing and prompt.
	selected := args[2:]

	if len(selected) == 0 {
		output, err := kubectl(sourceNS, "get", "secret", "-o", "jsonpath={range .items[*]}{.metadata.name}{'\\n'}{end}").Output()
		if err != nil {
			return err
		}

		var names []string
		for name := range strings.SplitSeq(strings.TrimSpace(string(output)), "\n") {
			if name = strings.TrimSpace(name); name != "" {
				names = append(names, name)
			}
		}

		if len(names) == 0 {
			fmt.Printf("No secrets found in namespace %q\n", sourceNS)
			return nil
		}

		fmt.Printf("Available secrets in %q:\n", sourceNS)
		for _, name := range names {
			fmt.Printf("  - %s\n", name)
		}
		fmt.Println()

		selection := Prompt(`Secret(s) to transfer (comma-separated names, or "all")`, PromptTypeString)

		if strings.EqualFold(strings.TrimSpace(selection), "all") {
			selected = names
		} else {
			for part := range strings.SplitSeq(selection, ",") {
				if name := strings.TrimSpace(part); name != "" {
					selected = append(selected, name)
				}
			}
		}

		if len(selected) == 0 {
			return fmt.Errorf("no secrets selected")
		}
	}

	for _, name := range selected {
		if err := transferSecret(sourceNS, targetNS, name); err != nil {
			return fmt.Errorf("transferring secret %q: %w", name, err)
		}
	}

	return nil
}

func transferSecret(sourceNS, targetNS, name string) error {
	output, err := kubectl(sourceNS, "get", "secret", name, "-o", "json").Output()
	if err != nil {
		return err
	}

	var secret map[string]any
	if err := json.Unmarshal(output, &secret); err != nil {
		return fmt.Errorf("parsing secret JSON: %w", err)
	}

	if metadata, ok := secret["metadata"].(map[string]any); ok {
		delete(metadata, "resourceVersion")
		delete(metadata, "uid")
		delete(metadata, "creationTimestamp")
		delete(metadata, "selfLink")
		delete(metadata, "managedFields")
		delete(metadata, "generation")
		metadata["namespace"] = targetNS

		if annotations, ok := metadata["annotations"].(map[string]any); ok {
			delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")
			if len(annotations) == 0 {
				delete(metadata, "annotations")
			}
		}
	}

	manifestJSON, err := json.Marshal(secret)
	if err != nil {
		return fmt.Errorf("marshaling secret: %w", err)
	}

	reader := strings.NewReader(string(manifestJSON))

	exists := kubectl(targetNS, "get", "secret", name, "-o", "name").Silent().Success()
	if exists {
		fmt.Printf("Replacing secret %q in namespace %q\n", name, targetNS)
		return kubectl(targetNS, "replace", "-f", "-").Stdin(reader).Run()
	}

	fmt.Printf("Creating secret %q in namespace %q\n", name, targetNS)
	return kubectl(targetNS, "create", "-f", "-").Stdin(reader).Run()
}

func parseKeyValue(kv string) (key, value string, err error) {
	idx := strings.Index(kv, "=")
	if idx == -1 {
		return "", "", fmt.Errorf("missing '=' in key=value pair")
	}

	key = kv[:idx]
	value = kv[idx+1:]

	if key == "" {
		return "", "", fmt.Errorf("key cannot be empty")
	}

	return key, value, nil
}
