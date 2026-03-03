package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	. "github.com/streamingfast/cli"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

var zlog, _ = logging.RootLogger("kcctx", "github.com/streamingfast/tooling/cmd/kcctx")

func init() {
	logging.InstantiateLoggers(logging.WithDefaultLevel(zap.ErrorLevel))
}

func main() {
	Run(
		"kcctx [-g] [<cluster>@]<namespace>",
		"Manages to which cluster/namespace your environment works with locally or globally",
		Description(`
			Switches your kubectl context to a specific cluster/namespace by generating
			a scoped kubeconfig file from a master config template.

			The tool outputs 'export KUBECONFIG=...' statements meant to be eval'd:

			  $(kcctx eth-mainnet)

			Prerequisites:

			1. A master kubeconfig at ~/.kube/master.config containing all your cluster
			   definitions, users and credentials. This file is used as a template and is
			   never modified. It should only contain 'clusters:' and 'users:' sections
			   (no 'contexts:' or 'current-context:' as those are managed by kcctx).

			   The easiest way to create it is to copy your existing kubeconfig and strip
			   the context-related fields:

			     cp ~/.kube/config ~/.kube/master.config

			   Then edit ~/.kube/master.config and remove the 'current-context:' line and
			   the entire 'contexts:' block, keeping only 'apiVersion', 'kind', 'clusters'
			   and 'users'. The result should look like:

			     apiVersion: v1
			     kind: Config
			     clusters:
			       - name: my-gke-cluster
			         cluster:
			           server: https://...
			           certificate-authority-data: ...
			     users:
			       - name: my-user
			         user:
			           exec: ...

			2. A kcctx config at ~/.config/kcctx/config.yaml that maps cluster names
			   to their user in the master config:

			     default_cluster: my-gke-cluster
			     clusters:
			       my-gke-cluster:
			         user: my-user
			       other-cluster:
			         name: actual-kube-cluster-name  # optional, if different from key
			         user: other-user

			When invoked, kcctx reads master.config, extracts only the relevant cluster
			and user entries, creates a context for <cluster>/<namespace>, and writes a
			scoped kubeconfig to ~/.kube/config-<cluster>-<namespace>.

			With -g (global), it writes directly to ~/.kube/config instead, affecting
			all terminals.
		`),
		ExactArgs(1),
		PersistentFlags(func(flags *pflag.FlagSet) {
			flags.BoolP("global", "g", false, "The environment changes will apply globally to your system affecting the other terminals or new ones created")
		}),
		Example(`
			# Configure your environment to use 'eth-mainnet' namespace on <default_cluster>
			$(kcctx eth-mainnet)
		`),
		Execute(execute),

		ConfigureViper("KCCTX"),
	)
}

func execute(cmd *cobra.Command, args []string) error {
	// FIXME: Ensure only of local or global flag is set

	input, err := ParseInput(args[0])
	if err != nil {
		return fmt.Errorf("invalid argument %q: %w", args[0], err)
	}

	defaultConfigLocation, err := DefaultConfigLocation()
	if err != nil {
		return fmt.Errorf("default config location: %w", err)
	}

	config, err := LoadConfig(defaultConfigLocation)
	if err != nil {
		return fmt.Errorf("unable to load config: %w", err)
	}

	zlog.Info("config loaded", zap.Reflect("config", config))

	if input.Cluster == "" && config.DefaultCluster == "" {
		return fmt.Errorf(`cannot use "<cluster>@<namespace>" invocation because "default_cluster" is not set in %q`, defaultConfigLocation)
	}

	kubeConfig, err := generateKubeConfig(config, input)
	if err != nil {
		return fmt.Errorf("generate kube config: %w", err)
	}

	kubeConfigDirectory, err := DefaultKubeConfigDirectoryLocation()
	if err != nil {
		return fmt.Errorf("unable to determine kube config directory: %w", err)
	}

	kubeMasterConfig, err := ParseKubeMasterConfig(kubeConfigDirectory)
	if err != nil {
		return fmt.Errorf("parse kube master config: %w", err)
	}

	kubeMasterConfig.SetActiveContext(kubeConfig.Name)
	kubeMasterConfig.SetContexts(kubeConfig)
	kubeMasterConfig.KeepOnlyClusterWithNameIn(kubeConfig.Context.Cluster)
	kubeMasterConfig.KeepOnlyUserWithNameIn(kubeConfig.Context.User)

	kubeConfigFile := filepath.Join(kubeConfigDirectory, "config-"+strings.ReplaceAll(kubeConfig.Name, "/", "-"))
	if viper.GetBool("global-global") {
		kubeConfigFile = filepath.Join(kubeConfigDirectory, "config")
	}

	zlog.Debug("storing kube context", zap.String("path", kubeConfigFile))
	if err := kubeMasterConfig.WriteTo(kubeConfigFile); err != nil {
		return fmt.Errorf("unable to write kube config file: %w", err)
	}

	fmt.Printf("export KUBECONFIG=%s\n", kubeConfigFile)
	fmt.Printf("export BULLETTRAIN_KCTX_KCONFIG=%s\n", kubeConfigFile)

	return nil
}

func generateKubeConfig(config *Config, input *Input) (*KubeConfig, error) {
	cluster := config.DefaultCluster
	if input.Cluster != "" {
		cluster = input.Cluster
	}

	clusterSpec := config.FindClusterSpec(cluster)
	if clusterSpec == nil {
		return nil, fmt.Errorf("no cluster named %q found in your config", cluster)
	}

	if clusterSpec.User == "" {
		return nil, fmt.Errorf("cluster spec named %q does not have a configured user associated with", cluster)
	}
	kubeCluster := cluster
	if clusterSpec.Name != "" {
		kubeCluster = clusterSpec.Name
	}

	return &KubeConfig{
		Name: fmt.Sprintf("%s/%s", cluster, input.Namespace),
		Context: &KubeContext{
			Cluster:   kubeCluster,
			Namespace: input.Namespace,
			User:      clusterSpec.User,
		},
	}, nil
}
