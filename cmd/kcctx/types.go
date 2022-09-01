package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

type Input struct {
	Cluster   string
	Namespace string
}

func ParseInput(in string) (*Input, error) {
	left, right, found := strings.Cut(in, "@")
	if !found {
		return &Input{Cluster: "", Namespace: left}, nil
	}

	return &Input{Cluster: left, Namespace: right}, nil
}

func DefaultKubeConfigDirectoryLocation() (string, error) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to determine user directory: %w", err)
	}

	return filepath.Join(userHome, ".kube"), nil
}

type KubeConfig struct {
	Name    string       `yaml:"name"`
	Context *KubeContext `yaml:"context"`
}

type KubeContext struct {
	Cluster   string `yaml:"cluster"`
	Namespace string `yaml:"namespace"`
	User      string `yaml:"user"`
}

type KubeMasterConfig struct {
	internal map[string]interface{}
}

func (c *KubeMasterConfig) SetActiveContext(context string) {
	c.internal["current-context"] = context
}

func (c *KubeMasterConfig) SetContexts(contexts ...*KubeConfig) {
	c.internal["contexts"] = contexts
}

func (c *KubeMasterConfig) KeepOnlyClusterWithNameIn(names ...string) {
	var filtered []interface{}
	for _, cluster := range c.internal["clusters"].([]interface{}) {
		cluster := cluster.(map[string]interface{})

		if slices.Contains(names, cluster["name"].(string)) {
			filtered = append(filtered, cluster)
		}
	}

	c.internal["clusters"] = filtered
}

func (c *KubeMasterConfig) KeepOnlyUserWithNameIn(names ...string) {
	users := c.internal["users"].([]interface{})
	zlog.Debug("filtering user not within our accepted names", zap.Int("user_count", len(users)), zap.Strings("accepted_names", names))

	var filtered []interface{}
	for _, user := range users {
		cluster := user.(map[string]interface{})

		if slices.Contains(names, cluster["name"].(string)) {
			filtered = append(filtered, cluster)
		}
	}

	c.internal["users"] = filtered
}

func (c *KubeMasterConfig) WriteTo(location string) error {
	content, err := yaml.Marshal(c.internal)
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}

	return os.WriteFile(location, content, os.ModePerm)
}

func DefaultKubeMasterConfigLocation() (string, error) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to determine user directory: %w", err)
	}

	return filepath.Join(userHome, ".kube", "master.config"), nil
}

func ParseKubeMasterConfig(kubeConfigDirectory string) (*KubeMasterConfig, error) {
	masterFile := filepath.Join(kubeConfigDirectory, "master.config")
	masterFileContent, err := os.ReadFile(masterFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read kube master config file %q: %w", masterFile, err)
	}

	config := &KubeMasterConfig{}

	if err := yaml.Unmarshal(masterFileContent, &config.internal); err != nil {
		return nil, fmt.Errorf("unable to unmarshal kube master config file %q: %w", masterFile, err)
	}

	return config, nil
}

type Config struct {
	// DefaultCluster is the name of the cluster to use when none is passed in the CLI command
	DefaultCluster string `yaml:"default_cluster"`

	// Clusters defines a list of clusters defined in master config specially to create the mapping
	// from cluster name to user.
	Clusters map[string]*ClusterSpec `yaml:"clusters"`
}

func (c *Config) FindClusterSpec(clusterName string) *ClusterSpec {
	for name, spec := range c.Clusters {
		if name == clusterName {
			return spec
		}
	}

	return nil
}

type ClusterSpec struct {
	// this corresponds to the name in the .kube/config, if it is empty we will use the key
	Name string `yaml:"name,omitempty"`
	User string `yaml:"user"`
}

func LoadConfig(file string) (*Config, error) {
	zlog.Debug("trying to load config file", zap.String("file", file))
	content, err := ioutil.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return newDefaultConfig(), nil
		}

		return nil, fmt.Errorf("unable to read config file %q: %w", file, err)
	}

	config := newDefaultConfig()
	err = yaml.Unmarshal(content, config)

	return config, err
}

func DefaultConfigLocation() (string, error) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to determine user directory: %w", err)
	}

	return filepath.Join(userHome, ".config", "kcctx", "config.yaml"), nil
}

func newDefaultConfig() *Config {
	return &Config{
		Clusters: map[string]*ClusterSpec{},
	}
}
