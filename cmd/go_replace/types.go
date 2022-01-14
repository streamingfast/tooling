package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/streamingfast/cli"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type Replacement struct {
	From string
	To   string
}

type UnresolvedReplacement string

var projectDepRegex = regexp.MustCompile(`^[^/@]+/[^/]+$`)
var versionSuffixRegex = regexp.MustCompile(`!$`)

func (u UnresolvedReplacement) Resolve(config *Config) (out Replacement, err error) {
	if config.DefaultWorkDir == "" {
		return out, fmt.Errorf("no configuration defined for work directory element, must specify a 'default_work_dir' config value")
	}

	from := strings.TrimSpace(string(u))
	to := ""

	if len(from) <= 1 {
		return out, fmt.Errorf("invalid package ID %q: not long enough", from)
	}

	// Only one of @ or ~ is accepted
	switch {
	case strings.HasPrefix(from, "@"):
		to, err = resolveWorkingDirPath(config.DefaultWorkDir, strings.Replace(from, "@", "", 1))
		from = replaceInPackageIDPrefix("@", config.DefaultRepoShortcut, from)

	case strings.HasPrefix(from, "~"):
		if config.DefaultProjectShortcut == "" {
			return out, fmt.Errorf("unable to resolve package ID %q: no configuration defined for ~ element, must specify a 'default_project_shortcut' config value", from)
		}

		to, err = resolveWorkingDirPath(config.DefaultWorkDir, strings.Replace(from, "~", "", 1))
		from = replaceInPackageIDPrefix("~", config.DefaultProjectShortcut, from)
	case looksLikeRelativePath(from):
		if config.DefaultProjectShortcut == "" {
			return out, fmt.Errorf("unable to resolve package ID %q: no configuration defined for a plain dependency, must specify a 'default_project_shortcut' config value", from)
		}

		to, err = resolveWorkingDirPath(".", from)
		from = prependToPackageID(config.DefaultProjectShortcut, filepath.Base(from))

	default:
		if config.DefaultProjectShortcut == "" {
			return out, fmt.Errorf("unable to resolve package ID %q: no configuration defined for a plain dependency, must specify a 'default_project_shortcut' config value", from)
		}

		to, err = resolveWorkingDirPath(config.DefaultWorkDir, from)
		from = prependToPackageID(config.DefaultProjectShortcut, from)
	}

	if err != nil {
		return out, fmt.Errorf("resolve working dir: %w", err)
	}

	return Replacement{from, to}, nil
}

func looksLikeRelativePath(dep string) bool {
	return strings.Contains(dep, ".") || strings.Contains(dep, string(os.PathSeparator))
}

func resolveWorkingDirPath(workingDir string, from string) (string, error) {
	path := filepath.Join(workingDir, from)
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("unable to make path %q absolute: %w", path, err)
	}

	if !cli.DirectoryExists(abs) {
		return "", fmt.Errorf("the replacement path %q does not exist", abs)
	}

	return abs, nil
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
	// DefaultWorkDir defines value where local dependency are resolved to
	DefaultWorkDir string `yaml:"default_work_dir"`

	// DefaultRepoShortcut defines value of leading @ in package id, usually github.com
	DefaultRepoShortcut string `yaml:"default_repo_shortcut"`

	// DefaultProjectShortcut defines value of leading ~ in package id, usually github.com/<org>
	DefaultProjectShortcut string `yaml:"default_project_shortcut"`
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
	if err == nil && config.DefaultWorkDir != "" {
		config.DefaultWorkDir = os.ExpandEnv(config.DefaultWorkDir)
	}

	return config, err
}

func newDefaultConfig() *Config {
	return &Config{
		DefaultRepoShortcut: "github.com/",
	}
}
