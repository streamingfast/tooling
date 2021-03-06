package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type PackageID string

type UnresolvedPackageID string

var versionSuffixRegex = regexp.MustCompile(`!$`)

func (u UnresolvedPackageID) Resolve(config *Config) (PackageID, error) {
	in := strings.TrimSpace(string(u))
	if len(in) <= 1 {
		return "", fmt.Errorf("invalid package ID %q: not long enough", in)
	}

	// Only one of @ or ~ is accepted
	switch {
	case strings.HasPrefix(in, "@"):
		in = replaceInPackageIDPrefix("@", config.DefaultRepoShortcut, in)

	case strings.HasPrefix(in, "~"):
		if config.DefaultProjectShortcut == "" {
			return "", fmt.Errorf("unable to resolve package ID %q: no configuration defined for ~ element, must specify a 'default_project_shortcut' config value", in)
		}

		in = replaceInPackageIDPrefix("~", config.DefaultProjectShortcut, in)
	}

	if versionSuffixRegex.MatchString(in) {
		in = versionSuffixRegex.ReplaceAllString(in, config.DefaultBranchShortcut)
	}

	return PackageID(in), nil
}

func replaceInPackageIDPrefix(symbol, replacement, rest string) string {
	suffix := strings.Replace(rest, symbol, "", 1)
	hasTrailingSlash := strings.HasSuffix(replacement, "/")
	hasLeadingSlash := strings.HasPrefix(suffix, "/")

	if !hasTrailingSlash && !hasLeadingSlash {
		return replacement + "/" + suffix
	}

	if (hasTrailingSlash && !hasLeadingSlash) || (!hasTrailingSlash && hasLeadingSlash) {
		return replacement + suffix
	}

	return replacement + strings.Replace(suffix, "/", "", 1)
}

type Config struct {
	// DefaultRepoShortcut defines value of leading @ in package id
	DefaultRepoShortcut string `yaml:"default_repo_shortcut"`

	// DefaultRepoShortcut defines value of leading ~ in package id
	DefaultProjectShortcut string `yaml:"default_project_shortcut"`

	// DefaultBranchShortcut defines value of trailing ! in package id
	DefaultBranchShortcut string `yaml:"default_branch_shortcut"`
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

func newDefaultConfig() *Config {
	return &Config{
		DefaultRepoShortcut:   "github.com/",
		DefaultBranchShortcut: "@develop",
	}
}
