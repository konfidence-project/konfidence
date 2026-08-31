package config

import (
	"context"
	encodingjson "encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	kdenapi "github.com/konfidence-project/konfidence/internal/kden/apiclient"
	"github.com/konfidence-project/konfidence/internal/kden/auth"
	"github.com/spf13/cobra"
)

type Configuration struct {
	LogLevel       string `json:"log-level" koanf:"log-level"`
	LogFormat      string `json:"log-format" koanf:"log-format"`
	Output         string `json:"output" koanf:"output"`
	APIEndpoint    string `json:"api-endpoint" koanf:"api-endpoint"`
	LoginTimeout   string `json:"login-timeout" koanf:"login-timeout"`
	RequestTimeout string `json:"request-timeout" koanf:"request-timeout"`
}

type AppConfig struct {
	APIProvider *APIClientProvider
}

type APIClientProvider struct {
	once       sync.Once
	authClient *auth.Client
	err        error
	factory    func() (*auth.Client, error)
}

func NewAPIClientProvider(
	factory func() (*auth.Client, error),
) *APIClientProvider {
	return &APIClientProvider{factory: factory}
}

func (p *APIClientProvider) AuthClient() (*auth.Client, error) {
	p.once.Do(func() {
		p.authClient, p.err = p.factory()
	})

	return p.authClient, p.err
}

type ProjectAPI interface {
	ListProjectsV1WithResponse(
		ctx context.Context,
		reqEditors ...kdenapi.RequestEditorFn,
	) (*kdenapi.ListProjectsV1Response, error)
}

type Authenticator interface {
	Login(context.Context) error
	Invalidate() error
}

type envVarConfigs struct {
	kdenEnvPrefix      string
	logLevelEnvName    string
	logFormatEnvName   string
	outputEnvName      string
	apiEndpointEnvName string
}

type configFileFunctions struct {
	getPath    func() (string, error)
	updateFile func(string, []byte) error
}

var (
	Config Configuration
	k      = koanf.New(".")
)

var (
	kdenEnvPrefix      = "KDEN_"
	logLevelEnvName    = "KDEN_LOG_LEVEL"
	outputEnvName      = "KDEN_OUTPUT"
	logFormatEnvName   = "KDEN_LOG_FORMAT"
	apiEndpointEnvName = "KDEN_API_ENDPOINT"
	RootCommandName    = "kden"
	envRegexPattern    = "^[A-Z]+(_[A-Z]+)*$"
)

var SupportedConfigurations = map[string][]string{
	"log-level":       {"debug", "info", "error"},
	"log-format":      {"json", "text", "pretty"},
	"output":          {"json", "yaml", "pretty"},
	"api-endpoint":    {},
	"login-timeout":   {},
	"request-timeout": {},
}

var configFileFuncs = configFileFunctions{
	getPath:    getOrCreateConfigFile,
	updateFile: updateConfigFile,
}

var envConfigs = envVarConfigs{
	kdenEnvPrefix:      kdenEnvPrefix,
	logLevelEnvName:    logLevelEnvName,
	logFormatEnvName:   logFormatEnvName,
	outputEnvName:      outputEnvName,
	apiEndpointEnvName: apiEndpointEnvName,
}

func Configure(cmd *cobra.Command) error {
	err := k.Load(confmap.Provider(map[string]interface{}{
		"log-level":       "info",
		"log-format":      "pretty",
		"output":          "json",
		"api-endpoint":    "http://localhost:8090/api",
		"login-timeout":   "2m",
		"request-timeout": "30s",
	}, "."), nil)
	if err != nil {
		return fmt.Errorf("failed to load default configuration: %w", err)
	}

	filePath, err := configFileFuncs.getPath()
	if err != nil {
		return fmt.Errorf("failed to get config file path: %w", err)
	}

	err = k.Load(file.Provider(filePath), json.Parser())
	if err != nil {
		return fmt.Errorf("failed to load configuration from file: %w", err)
	}

	err = k.Load(env.Provider(envConfigs.kdenEnvPrefix, ".", func(s string) string {
		re := regexp.MustCompile(envRegexPattern)
		if !re.MatchString(s) {
			return ""
		}

		s = strings.TrimPrefix(s, envConfigs.kdenEnvPrefix)
		s = strings.ReplaceAll(s, "_", "-")
		return strings.ToLower(s)
	}), nil)
	if err != nil {
		return fmt.Errorf("failed to load configuration from environment: %w", err)
	}

	err = k.Load(posflag.Provider(cmd.Flags(), ".", k), nil)
	if err != nil {
		return fmt.Errorf("failed to load configuration from flags: %w", err)
	}

	if err := k.Unmarshal("", &Config); err != nil {
		return fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	err = validateConfig(&Config)
	if err != nil {
		return fmt.Errorf("failed to validate configuration: %w", err)
	}

	return nil
}

func SetKey(key string, value string) error {
	configMap, err := structToMap(&Config)
	if err != nil {
		return fmt.Errorf("failed to convert config to map: %w", err)
	}

	_, ok := configMap[key]
	if !ok {
		return fmt.Errorf("'%s' is not a valid configuration key", key)
	}

	// Keys with an empty allowed-values list accept any non-empty value,
	// but may have additional validation (e.g. api-endpoint must be a valid URL).
	allowed := SupportedConfigurations[key]
	if len(allowed) > 0 && !slices.Contains(allowed, value) {
		return fmt.Errorf("value '%s' is not valid for configuration key '%s'. Supported values are: %s", value, key, strings.Join(allowed, ", ")) //nolint:lll
	}
	if value == "" {
		return fmt.Errorf("value for configuration key '%s' must not be empty", key)
	}
	if key == "api-endpoint" {
		if err := validateAPIEndpoint(value); err != nil {
			return err
		}
	}
	configMap[key] = value

	configFilePath, err := configFileFuncs.getPath()
	if err != nil {
		return fmt.Errorf("failed to get config file path: %w", err)
	}

	configAsBytes, err := encodingjson.Marshal(configMap)
	if err != nil {
		return fmt.Errorf("failed to marshal updated config: %w", err)
	}
	return configFileFuncs.updateFile(configFilePath, configAsBytes)
}

func UnSetKey(key string) error {
	configMap, err := structToMap(&Config)
	if err != nil {
		return fmt.Errorf("failed to convert config to map: %w", err)
	}
	_, ok := configMap[key]
	if !ok {
		return fmt.Errorf("'%s' is not a valid configuration key", key)
	}
	delete(configMap, key)

	configFilePath, err := configFileFuncs.getPath()
	if err != nil {
		return fmt.Errorf("failed to get config file path: %w", err)
	}
	configAsBytes, err := encodingjson.Marshal(configMap)
	if err != nil {
		return fmt.Errorf("failed to marshal config to JSON: %w", err)
	}

	return configFileFuncs.updateFile(configFilePath, configAsBytes)
}

func structToMap(s interface{}) (map[string]interface{}, error) {
	data, err := encodingjson.Marshal(s)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	err = encodingjson.Unmarshal(data, &result)
	return result, err
}

func validateConfig(cfg *Configuration) error {
	if !slices.Contains(SupportedConfigurations["log-level"], cfg.LogLevel) {
		return fmt.Errorf("invalid log-level: %s", cfg.LogLevel)
	}

	if !slices.Contains(SupportedConfigurations["log-format"], cfg.LogFormat) {
		return fmt.Errorf("invalid log-format: %s", cfg.LogFormat)
	}

	if !slices.Contains(SupportedConfigurations["output"], cfg.Output) {
		return fmt.Errorf("invalid output: %s", cfg.Output)
	}

	if cfg.APIEndpoint == "" {
		return fmt.Errorf("invalid api-endpoint: must not be empty")
	}

	if err := validateAPIEndpoint(cfg.APIEndpoint); err != nil {
		return err
	}

	loginTimeout, err := parsePositiveDuration(
		"login-timeout",
		cfg.LoginTimeout,
	)
	if err != nil {
		return err
	}

	requestTimeout, err := parsePositiveDuration(
		"request-timeout",
		cfg.RequestTimeout,
	)
	if err != nil {
		return err
	}

	if loginTimeout <= requestTimeout {
		return fmt.Errorf(
			"invalid timeouts: login-timeout %q must be greater than request-timeout %q",
			cfg.LoginTimeout,
			cfg.RequestTimeout,
		)
	}

	return nil
}

func validateAPIEndpoint(addr string) error {
	u, err := url.Parse(addr)
	if err != nil {
		return fmt.Errorf(
			"invalid api-endpoint %q: %w\n"+
				"  Set it with:  kden config set api-endpoint http://<host>:<port>\n"+
				"  Or via env:   KDEN_API_ENDPOINT=http://<host>:<port>",
			addr, err,
		)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf(
			"invalid api-endpoint %q: scheme must be http or https, got %q\n"+
				"  Set it with:  kden config set api-endpoint http://<host>:<port>\n"+
				"  Or via env:   KDEN_API_ENDPOINT=http://<host>:<port>",
			addr, u.Scheme,
		)
	}
	if u.Host == "" {
		return fmt.Errorf(
			"invalid api-endpoint %q: host must not be empty\n"+
				"  Set it with:  kden config set api-endpoint http://<host>:<port>\n"+
				"  Or via env:   KDEN_API_ENDPOINT=http://<host>:<port>",
			addr,
		)
	}
	return nil
}

func parsePositiveDuration(name, value string) (time.Duration, error) {
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s has an invalid value %q: %w", name, value, err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", name)
	}

	return duration, nil
}
