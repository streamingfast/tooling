package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type ResolvedInput struct {
	Organization string
	Repository   string
	Expanded     string
}

type UnresolvedInput string

var orgRepoRegex = regexp.MustCompile(`^([^/]+)/(.*?)(?:.git)?$`)
var githubOrgRepoRegex = regexp.MustCompile(`^github.com/([^/]+)/(.*?)(?:.git)?$`)
var httpsGithubOrgRepoRegex = regexp.MustCompile(`^https://github.com/([^/]+)/(.*?)(?:.git)?$`)
var sshGithubOrgRepoRegex = regexp.MustCompile(`^git@github.com:([^/]+)/(.*?)(?:.git)?$`)

type regexMatch struct {
	*regexp.Regexp
	id   string
	mode string
}

func (u UnresolvedInput) Resolve(config *Config) (out ResolvedInput, err error) {
	defer func() {
		zlog.Debug("resolved input", zap.String("unresolved", string(u)), zap.Reflect("resolved", out), zap.Error(err))
	}()

	in := strings.TrimSpace(string(u))
	if len(in) <= 1 {
		return out, fmt.Errorf("invalid input %q: not long enough", in)
	}

	var regexMatchers []regexMatch = []regexMatch{
		// From most specific to least specific!
		{Regexp: sshGithubOrgRepoRegex, id: "git@github.com:org/repo", mode: "ssh"},
		{Regexp: httpsGithubOrgRepoRegex, id: "https://github.com/org/repo", mode: "https"},
		{Regexp: githubOrgRepoRegex, id: "github.com/org/repo", mode: "https"},
		{Regexp: orgRepoRegex, id: "org/repo", mode: "https"},
	}

	for _, regexMatch := range regexMatchers {
		if groups := regexMatch.FindStringSubmatch(in); len(groups) > 0 {
			zlog.Debug(fmt.Sprintf("expanding %s shortcut", regexMatch.id), zap.Strings("groups", groups))
			if len(groups) != 3 {
				return out, fmt.Errorf("invalid input %q: invalid %s format", in, regexMatch.id)
			}

			out.Organization = groups[1]
			out.Repository = groups[2]

			if regexMatch.mode == "https" {
				out.Expanded = fmt.Sprintf("https://github.com/%s/%s", out.Organization, out.Repository)
			} else {
				out.Expanded = fmt.Sprintf("git@github.com:%s/%s", out.Organization, out.Repository)
			}

			if !strings.HasSuffix(out.Expanded, ".git") {
				out.Expanded += ".git"
			}

			return out, nil
		}
	}

	return out, fmt.Errorf("invalid input %q: unknown format", in)
}

type Config struct {
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

	config := newDefaultConfig()
	err = yaml.Unmarshal(content, config)

	return config, err
}

func newDefaultConfig() *Config {
	return &Config{}
}
