package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/streamingfast/cli"
	. "github.com/streamingfast/cli"
	"github.com/streamingfast/cli/sflags"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

var zlog, _ = logging.RootLogger("kcctx", "github.com/streamingfast/tooling/cmd/kcctx")

func init() {
	logging.InstantiateLoggers(logging.WithDefaultLevel(zap.ErrorLevel))
}

func main() {
	Run(
		"sfenv <identifier>",
		"Exports various environment variables to configure your StreamingFast API access (API Keys, JWT, network, etc.)",
		Description(`
			You specify a configuration file at ~/.config/sfenv/config.yaml with the following structure:

			apiKeys:
			  primary: server_key1
			  default: "@primary"
			networks:
			  'default':
			    endpoint: mainnet.eth.streamingfast.io:443
			  'eth-mainnet':
			    endpoint: mainnet.eth.streamingfast.io:443
			    apiKey: @primary
			    alias:
			    - mainnet
			  'payment-gateway':
			    endpoint: payment.gateway.streamingfast.io:443
			    apiKey: server_key2
			    alias:
			    - pg

			You can then use:

			$(sfenv)
			# Env definition SF_API_KEY=server_key1, SF_API_TOKEN='jwt server_key1', SF_ENDPOINT=mainnet.eth.streamingfast.io:443

			$(sfenv eth-sepolia)
			# Env definition SF_API_KEY=server_key1, SF_API_TOKEN='jwt server_key1', SF_ENDPOINT=sepolia.eth.streamingfast.io:443

			$(sfenv pg)
			# Env definition SF_API_KEY=server_key2, SF_API_TOKEN='jwt server_key2', SF_ENDPOINT=payment.gateway.streamingfast.io:443

			You get variations with SUBSTREAMS_, DFUSE_ and STREAMINGFAST_FAST_ prefixes too (instead of 'SF_').

			JWT token are cached to disk in ~/.config/sfenv/jwt-cache/ and refreshed only if expired or if --refresh (-r) flag is passed:

			$(sfenv -r eth-sepolia)
			# Force JWT refresh, useful if you need new features set on your key
			`),
		MinimumNArgs(0),
		MaximumNArgs(1),
		PersistentFlags(func(flags *pflag.FlagSet) {
			flags.BoolP("refresh", "r", false, "Force refresh JWT token from the auth server if already present in cache")
		}),
		AfterAllHook(func(cmd *cobra.Command) {
			cli.ConfigureViperForCommand(cmd, "SFENV")
		}),
		Execute(execute),
	)
}

func execute(cmd *cobra.Command, args []string) error {
	zlog.Info("sfenv command started", zap.Strings("args", args))

	input := &Input{}
	if len(args) > 0 {
		var err error
		input, err = ParseInput(args[0])
		if err != nil {
			return fmt.Errorf("invalid argument %q: %w", args[0], err)
		}
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

	endpoint := ""
	jwtIssuerBaseURL := "https://auth.streamingfast.io"
	apiKey := config.DefaultApiKey

	if config.DefaultNetwork != nil {
		endpoint = config.DefaultNetwork.Endpoint
		jwtIssuerBaseURL = config.DefaultNetwork.JWTIssuerBaseURL

		if key := config.DefaultNetwork.ApiKey; key != nil {
			key = apiKey
		}
	}

	if input.Identifier != "" {
		if input.IsEndpoint {
			endpoint = input.Identifier
		} else {
			if network := config.SearchNetwork(input.Identifier); network != nil {
				endpoint = network.Endpoint
				jwtIssuerBaseURL = network.JWTIssuerBaseURL

				if network.ApiKey != nil {
					apiKey = network.ApiKey
				}
			}
		}
	}

	if apiKey != nil {
		putsEnvVar("API_KEY", apiKey.Key)

		token, err := getToken(apiKey, jwtIssuerBaseURL, sflags.MustGetBool(cmd, "refresh"))
		if err != nil {
			return fmt.Errorf("getting token: %w", err)
		}

		putsEnvVar("API_TOKEN", token)
		if err := expandTokenFeatures(token); err != nil {
			return fmt.Errorf("expanding token features: %w", err)
		}
	}

	if endpoint != "" {
		putsEnvVar("ENDPOINT", endpoint)
	}

	return nil
}

var envPrefixes = []string{"SF", "FIREHOSE", "SUBSTREAMS", "STREAMINGFAST_FAST"}

func putsEnvVar(name, value string) {
	for _, prefix := range envPrefixes {
		fmt.Printf("export %s_%s=%s\n", prefix, strings.ToUpper(name), value)
	}
}

func expandTokenFeatures(token string) error {
	claims, err := ParseJWTUnverified(token)
	if err != nil {
		return fmt.Errorf("parsing JWT token: %w", err)
	}

	for key, val := range claims {
		if key == "cfg" {
			for key, value := range val.(map[string]any) {
				putsEnvVar("API_FEATURE_"+strings.ToUpper(key), fmt.Sprintf("%s", value))
			}
		}
	}

	return nil
}

func getToken(apiKey *ApiKey, jwtIssuerBaseURL string, forceRefresh bool) (string, error) {
	defaultJWTCache, err := DefaultJWTCacheLocation()
	if err != nil {
		return "", fmt.Errorf("default JWT cache location: %w", err)
	}

	var token *string
	if !forceRefresh {
		tokenOnDisk, err := tokenRead(apiKey, defaultJWTCache)
		if err != nil {
			return "", fmt.Errorf("reading token: %w", err)
		}

		if tokenOnDisk != nil {
			claims, err := ParseJWTUnverified(*tokenOnDisk)
			if err != nil {
				return "", fmt.Errorf("parsing JWT token: %w", err)
			}

			value, err := claims.GetExpirationTime()
			if err != nil {
				return "", fmt.Errorf("retrieve expiration time")
			}

			token = tokenOnDisk
			if value != nil && time.Now().After(value.Time) {
				zlog.Debug("token is expired, forcing refresh")
				token = nil
			}
		}
	} else {
		zlog.Debug("the JWT token was asked to be refreshed")
	}

	if token == nil {
		refreshedToken, err := tokenFetch(apiKey.Key, jwtIssuerBaseURL)
		if err != nil {
			return "", fmt.Errorf("fetching token: %w", err)
		}

		if err := tokenWrite(refreshedToken, apiKey, defaultJWTCache); err != nil {
			return "", fmt.Errorf("writing token: %w", err)
		}

		token = &refreshedToken
	}

	if token == nil {
		return "", fmt.Errorf("couldn't get a token")
	}

	return *token, nil
}

func tokenRead(apiKey *ApiKey, cacheFolder string) (*string, error) {
	tokenFile := tokenCacheJWTFile(apiKey, cacheFolder)
	zlog.Debug("reading token from cache", zap.String("api_key", apiKey.Name), zap.String("token_file", tokenFile), zap.String("cache_folder", cacheFolder))

	data, err := os.ReadFile(tokenFile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}

		return nil, fmt.Errorf("reading token cache file: %w", err)
	}

	token := string(data)
	return &token, nil
}

func tokenWrite(token string, apiKey *ApiKey, cacheFolder string) error {
	if err := os.MkdirAll(cacheFolder, 0700); err != nil {
		return fmt.Errorf("creating JWT cache directory: %w", err)
	}

	err := os.WriteFile(tokenCacheJWTFile(apiKey, cacheFolder), []byte(token), 0600)
	if err != nil {
		return fmt.Errorf("reading token cache file: %w", err)
	}

	return nil
}

func tokenCacheJWTFile(apiKey *ApiKey, cacheFolder string) string {
	return filepath.Join(cacheFolder, fmt.Sprintf("%s.jwt.txt", apiKey.Name))
}

func tokenFetch(apiKey string, jwtIssueBaseURL string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	data := fmt.Sprintf(`{"api_key":"%s"}`, apiKey)
	body := strings.NewReader(data)

	request, error := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/v1/auth/issue", jwtIssueBaseURL), body)
	if error != nil {
		return "", fmt.Errorf("new HTTP POST request: %w", error)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("HTTP POST request: %w", err)
	}
	defer response.Body.Close()

	fullBody, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("reading response body: %w", err)
	}

	if response.StatusCode != 200 {
		return "", fmt.Errorf("the HTTP POST request failed with code %d and body %q", response.StatusCode, fullBody)
	}

	var authResponse authResponse
	if err := json.Unmarshal(fullBody, &authResponse); err != nil {
		return "", fmt.Errorf("unmarshalling response %q: %w", fullBody, err)
	}

	return authResponse.Token, nil
}

type authResponse struct {
	Token string `json:"token"`
}
