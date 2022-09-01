package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/streamingfast/cli"
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
			TBW
		`),
		ExactArgs(1),
		PersistentFlags(func(flags *pflag.FlagSet) {
			flags.BoolP("local", "l", true, "The environment changes will apply locally to this instance without affecting the other terminals or new ones created")
			flags.BoolP("global", "g", false, "The environment changes will apply globally to your system affecting the other terminals or new ones created")
		}),
		AfterAllHook(func(cmd *cobra.Command) {
			cli.ConfigureViperForCommand(cmd, "KCCTX")
		}),
		Example(`
			# Configure your environment to use 'eth-mainnet' namespace on <default_cluster>
			$(kcctx eth-mainnet)
		`),
		Execute(execute),
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

	kubeConfigFile := filepath.Join(kubeConfigDirectory, "config")
	if viper.GetBool("global-local") {
		kubeConfigFile = filepath.Join(kubeConfigDirectory, "config-"+strings.ReplaceAll(kubeConfig.Name, "/", "-"))
	}

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
