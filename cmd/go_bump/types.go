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

var projectDepRegex = regexp.MustCompile(`^[^/@]+/[^/]+$`)
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
	case isPlainDependency(in):
		// Seems to be a plain dependency of the form "test"
		if config.DefaultProjectShortcut == "" {
			return "", fmt.Errorf("unable to resolve package ID %q: no configuration defined for a plain dependency, must specify a 'default_project_shortcut' config value", in)
		}

		in = prependToPackageID(config.DefaultProjectShortcut, in)

	case isProjectDependency(in):
		// Seems to be a project dependency of the form "<org>/test"
		if config.DefaultProjectShortcut == "" {
			return "", fmt.Errorf("unable to resolve package ID %q: no configuration defined for a plain dependency, must specify a 'default_project_shortcut' config value", in)
		}

		in = prependToPackageID(config.DefaultProjectShortcut, in)

	}

	if versionSuffixRegex.MatchString(in) {
		in = versionSuffixRegex.ReplaceAllString(in, config.DefaultBranchShortcut)
	}

	if !strings.Contains(in, "@") {
		in = in + config.DefaultBranchShortcut
	}

	return PackageID(in), nil
}

func isProjectDependency(dep string) bool {
	if !strings.Contains(dep, ".") && projectDepRegex.MatchString(dep) {
		return true
	}

	return false
}

func isPlainDependency(dep string) bool {
	if strings.Contains(dep, ".") {
		return false
	}

	if !strings.Contains(dep, ".") && !strings.Contains(dep, "/") {
		return true
	}

	return false
}

func replaceInPackageIDPrefix(symbol, replacement, in string) string {
	return prependToPackageID(replacement, strings.Replace(in, symbol, "", 1))
}

func prependToPackageID(prefix, in string) string {
	hasTrailingSlash := strings.HasSuffix(prefix, "/")
	hasLeadingSlash := strings.HasPrefix(in, "/")

	if !hasTrailingSlash && !hasLeadingSlash {
		return prefix + "/" + in
	}

	if (hasTrailingSlash && !hasLeadingSlash) || (!hasTrailingSlash && hasLeadingSlash) {
		return prefix + in
	}

	return prefix + strings.Replace(in, "/", "", 1)
}

type Config struct {
	// DefaultRepoShortcut defines value of leading @ in package id, usually github.com
	DefaultRepoShortcut string `yaml:"default_repo_shortcut"`

	// DefaultProjectShortcut defines value of leading ~ in package id, usually github.com/<org>
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
