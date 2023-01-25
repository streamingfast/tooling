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
		zlog.Debug("start with repo shortcut @")

		in = replaceInPackageIDPrefix("@", config.DefaultRepoShortcut, in)

	case strings.HasPrefix(in, "~"):
		zlog.Debug("start with project shortcut ~")

		if config.DefaultProjectShortcut == "" {
			return "", fmt.Errorf("unable to resolve package ID %q: no configuration defined for ~ element, must specify a 'default_project_shortcut' config value", in)
		}

		in = replaceInPackageIDPrefix("~", config.DefaultProjectShortcut, in)
	case isPlainDependency(in):
		// Seems to be a plain dependency of the form "test"
		zlog.Debug("inferred as plain dependency")

		if config.DefaultProjectShortcut == "" {
			return "", fmt.Errorf("unable to resolve package ID %q: no configuration defined for a plain dependency, must specify a 'default_project_shortcut' config value", in)
		}

		in = prependToPackageID(config.DefaultProjectShortcut, in)

	case isProjectDependency(in):
		zlog.Debug("inferred as project dependency")

		// Seems to be a project dependency of the form "<org>/test"
		if config.DefaultProjectShortcut == "" {
			return "", fmt.Errorf("unable to resolve package ID %q: no configuration defined for a plain dependency, must specify a 'default_project_shortcut' config value", in)
		}

		in = prependToPackageID(config.DefaultProjectShortcut, in)
	default:
		zlog.Debug("unable to infer anything about input")
	}

	if versionSuffixRegex.MatchString(in) {
		in = versionSuffixRegex.ReplaceAllString(in, config.DefaultBranchShortcut)
	}

	if !strings.Contains(in, "@") {
		in = in + config.DefaultBranchShortcut
	}

	resolved := PackageID(in)
	zlog.Debug("resolved dependency", zap.String("unresolved", string(u)), zap.String("resolved", string(resolved)))

	return resolved, nil
}

func isProjectDependency(dep string) bool {
	if !strings.Contains(dep, ".") && projectDepRegex.MatchString(dep) {
		return true
	}

	return false
}

func isPlainDependency(dep string) bool {
	// We might have "<package>" or "<package>@<version>", in both case, the 'strings.Cut'
	// left value will always be '<package>', so we can ignore the two other argument.
	pkg, _, _ := strings.Cut(dep, "@")

	if strings.Contains(pkg, ".") {
		return false
	}

	if !strings.Contains(pkg, ".") && !strings.Contains(pkg, "/") {
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

	// AfterBump specify some commands to run successful bump
	AfterBump *AfterBump `yaml:"after_bump"`
}

type AfterBump struct {
	GoModTidy bool `yaml:"go_mod_tidy"`
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
		AfterBump:             &AfterBump{GoModTidy: false},
	}
}
