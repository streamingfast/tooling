package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/exp/maps"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type Input struct {
	Identifier string
	IsEndpoint bool
}

func ParseInput(in string) (*Input, error) {
	return &Input{Identifier: in, IsEndpoint: strings.Contains(in, ".")}, nil
}

type Config struct {
	ApiKeys       []*ApiKey
	ApiKeysByName map[string]*ApiKey

	Networks        []*Network
	NetworksByName  map[string]*Network
	NetworksByAlias map[string]*Network

	DefaultApiKey  *ApiKey
	DefaultNetwork *Network
}

func (c *Config) JoinedApiKeyNames(sep string) string {
	return strings.Join(maps.Keys(c.ApiKeysByName), sep)
}

func (c *Config) SearchNetwork(input string) *Network {
	if network, found := c.NetworksByName[input]; found {
		return network
	}

	if network, found := c.NetworksByAlias[input]; found {
		return network
	}

	return nil
}

type ApiKey struct {
	Name string
	Key  string
}

type Network struct {
	Name     string
	Endpoint string
	Aliases  []string
	ApiKey   *ApiKey
}

func LoadConfig(file string) (*Config, error) {
	zlog.Debug("trying to load config file", zap.String("file", file))
	content, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return newDefaultConfig(), nil
		}

		return nil, fmt.Errorf("unable to read config file %q: %w", file, err)
	}

	parsed := &config{}
	err = yaml.Unmarshal(content, parsed)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal config file %q: %w", file, err)
	}

	config := &Config{
		ApiKeys:         make([]*ApiKey, 0, len(parsed.ApiKeys)),
		ApiKeysByName:   map[string]*ApiKey{},
		Networks:        make([]*Network, 0, len(parsed.Networks)),
		NetworksByName:  map[string]*Network{},
		NetworksByAlias: map[string]*Network{},
	}

	for apiKeyName, apiKeyValue := range parsed.ApiKeys {
		if strings.EqualFold(apiKeyName, "default") {
			// Ignore the default api key
			continue
		}

		config.ApiKeys = append(config.ApiKeys, &ApiKey{Name: apiKeyName, Key: apiKeyValue})
	}

	for _, apiKey := range config.ApiKeys {
		config.ApiKeysByName[apiKey.Name] = apiKey
	}

	if defaultApiKeyReference, found := parsed.ApiKeys["default"]; found {
		referencedName := strings.TrimPrefix(defaultApiKeyReference, "@")
		matchedKey, found := config.ApiKeysByName[referencedName]
		if !found {
			return nil, fmt.Errorf("unable to find default reference %q, valid api key names are [%q]", referencedName, config.JoinedApiKeyNames(", "))
		}

		config.DefaultApiKey = matchedKey
	}

	for networkName, networkConfig := range parsed.Networks {
		if strings.EqualFold(networkName, "default") {
			// Ignore the default api key
			continue
		}

		apiKey := config.DefaultApiKey
		if networkConfig.ApiKey != nil {
			apiKeyReference := strings.TrimPrefix(*networkConfig.ApiKey, "@")
			key, found := config.ApiKeysByName[apiKeyReference]
			if !found {
				return nil, fmt.Errorf("unable to find api key %q for network %q, valid api key names are [%q]", apiKeyReference, networkName, config.JoinedApiKeyNames(", "))
			}

			apiKey = key
		}

		network := &Network{Name: networkName, ApiKey: apiKey, Aliases: networkConfig.Aliases}
		if networkConfig.Endpoint != nil {
			network.Endpoint = *networkConfig.Endpoint
		}

		config.Networks = append(config.Networks, network)
		config.NetworksByName[networkName] = network
		for _, alias := range network.Aliases {
			config.NetworksByAlias[alias] = network
		}
	}

	if networkConfig, found := parsed.Networks["default"]; found {
		apiKey := config.DefaultApiKey
		if networkConfig.ApiKey != nil {
			key, found := config.ApiKeysByName[*networkConfig.ApiKey]
			if !found {
				return nil, fmt.Errorf("unable to find api key %q for default network, valid api key names are [%q]", *networkConfig.ApiKey, config.JoinedApiKeyNames(", "))
			}

			apiKey = key
		}

		defaultNetwork := &Network{Name: "default", ApiKey: apiKey}
		if networkConfig.Endpoint != nil {
			defaultNetwork.Endpoint = *networkConfig.Endpoint
		}

		config.DefaultNetwork = defaultNetwork
		config.NetworksByName["default"] = defaultNetwork
	}

	return config, err
}

type config struct {
	ApiKeys  map[string]string            `yaml:"apiKeys"`
	Networks map[string]*networkConfigDef `yaml:"networks"`
}

type networkConfigDef struct {
	Endpoint *string  `yaml:"endpoint"`
	Aliases  []string `yaml:"alias"`
	ApiKey   *string  `yaml:"apiKey"`
}

func DefaultConfigLocation() (string, error) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to determine user directory: %w", err)
	}

	return filepath.Join(userHome, ".config", "sfenv", "config.yaml"), nil
}

func DefaultJWTCacheLocation() (string, error) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to determine user directory: %w", err)
	}

	return filepath.Join(userHome, ".config", "sfenv", "jwt-cache"), nil
}

func newDefaultConfig() *Config {
	return &Config{
		ApiKeys:        []*ApiKey{},
		ApiKeysByName:  map[string]*ApiKey{},
		Networks:       []*Network{},
		NetworksByName: map[string]*Network{},

		DefaultApiKey:  nil,
		DefaultNetwork: nil,
	}
}
