package config

import (
	"errors"
	"fmt"
	"github.com/mitchellh/hashstructure/v2"
	"github.com/rs/zerolog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	MaxBrowserInstances                           = "MAX_BROWSER_INSTANCES"
	MaxBrowserInstancesDefault                    = 10
	DefaultChromeProfile                          = "DEFAULT_CHROME_PROFILE"
	DefaultChromeProfileDefault                   = ""
	ThroughputScaleUpThreshold                    = "THROUGHPUT_SCALE_UP_THRESHOLD"
	ThroughputScaleUpThresholdDefault             = 0.6
	MinBrowserInstances                           = "MIN_BROWSER_INSTANCES"
	MinBrowserInstancesDefault                    = 0
	EnableAutoAssignDebugPort                     = "ENABLE_AUTO_ASSIGN_DEBUG_PORT"
	EnableAutoAssignDebugPortDefault              = true
	EnableBrowserReuse                            = "ENABLE_BROWSER_REUSE"
	EnableBrowserReuseDefault                     = false
	ChromeDebugPorts                              = "CHROME_DEBUG_PORTS"
	ChromeHeadless                                = "CHROME_HEADLESS"
	ChromeHeadlessDefault                         = true
	ChromeEnableCustomProfiles                    = "CHROME_ENABLE_CUSTOM_PROFILES"
	ChromeEnableCustomProfilesDefault             = false
	ChromeEnableBrowserAutoShutdown               = "CHROME_ENABLE_BROWSER_AUTO_SHUTDOWN"
	ChromeEnableBrowserAutoShutdownDefault        = true
	ChromeBrowserAutoShutdownTimeoutInSecs        = "CHROME_BROWSER_AUTO_SHUTDOWN_TIMEOUT_IN_SECS"
	ChromeBrowserAutoShutdownTimeoutInSecsDefault = 30
	ChromeBrowserAutoIdleTimeoutInSecs            = "CHROME_BROWSER_AUTO_IDLE_TIMEOUT_IN_SECS"
	ChromeBrowserAutoIdleTimeoutInSecsDefault     = 30
	LogLevel                                      = "LOG_LEVEL"
	LogLevelDefault                               = zerolog.InfoLevel
	LogFilePath                                   = "LOG_OUTPUT"
	ServerPort                                    = "SERVER_PORT"
	ServerPortDefault                             = 3000
	ServerAccessToken                             = "SERVER_ACCESS_TOKEN"
	ServerAccessTokenDefault                      = ""
	ServerAccessTokenValidationEnabled            = "SERVER_ACCESS_TOKEN_VALIDATION_ENABLED"
	ServerAccessTokenValidationEnabledDefault     = false
	MaxCreateBrowserRetries                       = 20
	CreateBrowserRetrySleepInMs                   = 500
	StatsiteSink                                  = "STATSITE_SINK"
	StatsDSink                                    = "STATSD_SINK"
	DataDogHostName                               = "DATADOG_HOST"
	DataDogAddress                                = "DATADOG_ADDRESS"
)

type IConfig interface {
	GetChromeConfig() ChromeConfig
	GetChromePoolConfig() ChromePoolConfig
	GetLoggerConfig() LoggerConfig
	GetServerConfig() ServerConfig
	GetProxyQueueConfig() ProxyQueueConfig
	GetMetricsConfig() MetricsConfig
	Validate() error
}

type Config struct {
	chromePoolConfig ChromePoolConfig
	chromeConfig     ChromeConfig
	loggerConfig     LoggerConfig
	serverConfig     ServerConfig
	proxyQueueConfig ProxyQueueConfig
	metricsConfig    MetricsConfig
}

type MetricsConfig struct {
	StatsiteSink    string
	StatsDSink      string
	DataDogHostName string
	DataDogAddress  string
}

type ProxyQueueConfig struct {
	ThroughputScaleUpThreshold float64
}

type ChromeConfigOptionsPayload struct {
	Profile string
}

type ChromeConfigOptions struct {
	Profile string
	Hash    string
}

type ChromeConfig struct {
	EnableBrowserReuse               bool
	BrowserAutoSetIdleTimeoutInSecs  time.Duration
	Headless                         bool
	EnableTaggedBrowserAutoShutdown  bool
	EnableCustomChromeProfiles       bool
	EnableBrowserAutoShutdown        bool
	BrowserAutoShutdownTimeoutInSecs time.Duration
	DefaultOptions                   ChromeConfigOptions
}

type ChromePoolConfig struct {
	MaxBrowserInstances         int
	MinBrowserInstances         int
	EnableAutoAssignDebugPort   bool
	DebugPorts                  []int
	MaxCreateBrowserRetries     int
	CreateBrowserRetrySleepInMs int
}

type LoggerConfig struct {
	LogLevel    zerolog.Level
	LogFilePath string
}

type ServerConfig struct {
	Port                         int
	AccessToken                  string
	AccessTokenValidationEnabled bool
}

// Once - ONLY REFERENCE IN TESTS
var Once sync.Once

var c Config

func Get() IConfig {
	Once.Do(func() {
		c = Config{
			chromePoolConfig: ChromePoolConfig{
				MaxBrowserInstances:         getIntFromEnv(MaxBrowserInstances, MaxBrowserInstancesDefault),
				MinBrowserInstances:         getIntFromEnv(MinBrowserInstances, MinBrowserInstancesDefault),
				EnableAutoAssignDebugPort:   getBoolFromEnv(EnableAutoAssignDebugPort, EnableAutoAssignDebugPortDefault),
				MaxCreateBrowserRetries:     MaxCreateBrowserRetries,
				CreateBrowserRetrySleepInMs: CreateBrowserRetrySleepInMs,
				DebugPorts:                  getIntArrayFromEnv(ChromeDebugPorts, make([]int, 0)),
			},
			chromeConfig: ChromeConfig{
				Headless:                         getBoolFromEnv(ChromeHeadless, ChromeHeadlessDefault),
				EnableBrowserAutoShutdown:        getBoolFromEnv(ChromeEnableBrowserAutoShutdown, ChromeEnableBrowserAutoShutdownDefault),
				EnableCustomChromeProfiles:       getBoolFromEnv(ChromeEnableCustomProfiles, ChromeEnableCustomProfilesDefault),
				EnableBrowserReuse:               getBoolFromEnv(EnableBrowserReuse, EnableBrowserReuseDefault),
				BrowserAutoShutdownTimeoutInSecs: getSecTimeDurationFromEnv(ChromeBrowserAutoShutdownTimeoutInSecs, ChromeBrowserAutoShutdownTimeoutInSecsDefault),
				BrowserAutoSetIdleTimeoutInSecs:  getSecTimeDurationFromEnv(ChromeBrowserAutoIdleTimeoutInSecs, ChromeBrowserAutoIdleTimeoutInSecsDefault),
			},
			loggerConfig: LoggerConfig{
				LogLevel:    getLogLevelFromEnv(LogLevel, LogLevelDefault),
				LogFilePath: getStringFromEnv(LogFilePath, ""),
			},
			metricsConfig: MetricsConfig{
				StatsiteSink:    getStringFromEnv(StatsiteSink, ""),
				StatsDSink:      getStringFromEnv(StatsDSink, ""),
				DataDogHostName: getStringFromEnv(DataDogHostName, ""),
				DataDogAddress:  getStringFromEnv(DataDogAddress, ""),
			},
			serverConfig: ServerConfig{
				Port:                         getIntFromEnv(ServerPort, ServerPortDefault),
				AccessToken:                  getStringFromEnv(ServerAccessToken, ServerAccessTokenDefault),
				AccessTokenValidationEnabled: getBoolFromEnv(ServerAccessTokenValidationEnabled, ServerAccessTokenValidationEnabledDefault),
			},
			proxyQueueConfig: ProxyQueueConfig{
				ThroughputScaleUpThreshold: getFloat64FromEnv(ThroughputScaleUpThreshold, ThroughputScaleUpThresholdDefault),
			},
		}

		// TODO: handle error here
		defaultOpts, _ := NewCreateOptions(&ChromeConfigOptionsPayload{
			Profile: getStringFromEnv(DefaultChromeProfile, DefaultChromeProfileDefault),
		})
		c.chromeConfig.DefaultOptions = defaultOpts
	})
	return &c
}

func (c *Config) GetChromePoolConfig() ChromePoolConfig {
	return c.chromePoolConfig
}

func (c *Config) GetChromeConfig() ChromeConfig {
	return c.chromeConfig
}

func (c *Config) GetLoggerConfig() LoggerConfig {
	return c.loggerConfig
}

func (c *Config) GetServerConfig() ServerConfig {
	return c.serverConfig
}

func (c *Config) GetProxyQueueConfig() ProxyQueueConfig {
	return c.proxyQueueConfig
}

func (c *Config) GetMetricsConfig() MetricsConfig {
	return c.metricsConfig
}

func (c *Config) Validate() error {
	var errs []string

	// Validate ChromeDebugPorts are in min and max range
	if !c.chromePoolConfig.EnableAutoAssignDebugPort {
		if len(c.chromePoolConfig.DebugPorts) == 0 {
			errs = append(errs, fmt.Sprintf("if %s is disabled, %s is required", EnableAutoAssignDebugPort, ChromeDebugPorts))
		} else if len(c.chromePoolConfig.DebugPorts) < c.chromePoolConfig.MaxBrowserInstances {
			errs = append(errs, fmt.Sprintf("if %s is disabled, %s must contain %s number of ports", EnableAutoAssignDebugPort, ChromeDebugPorts, MaxBrowserInstances))
		}
	}

	if c.chromePoolConfig.MinBrowserInstances < 0 {
		errs = append(errs, fmt.Sprintf("%s must be greater than 0 if set", MinBrowserInstances))
	} else if c.chromePoolConfig.MaxBrowserInstances <= 0 {
		errs = append(errs, fmt.Sprintf("%s must be greater than or equal to 1 if set", MaxBrowserInstances))
	}

	if c.serverConfig.AccessTokenValidationEnabled && len(c.serverConfig.AccessToken) == 0 {
		errs = append(errs, fmt.Sprintf("%s is required if %s is enabled", ServerAccessToken, ServerAccessTokenValidationEnabled))
	}

	if c.proxyQueueConfig.ThroughputScaleUpThreshold <= 0 {
		errs = append(errs, fmt.Sprintf("%s must be greater than 0.0", ThroughputScaleUpThreshold))
	}

	if len(errs) > 0 {
		return errors.New(fmt.Sprintf("environment config failed validation with the following errors: \n%s", strings.Join(errs, ",\n")))
	}

	return nil
}

func getEnvValByKey(envKey string) (string, bool) {
	ev := os.Getenv(envKey)
	return ev, len(ev) > 0
}

func getBoolFromEnv(envKey string, defaultVal bool) bool {
	ev, exists := getEnvValByKey(envKey)
	if !exists {
		return defaultVal
	}
	envValBool, err := strconv.ParseBool(ev)
	if err != nil {
		// TODO: log warning
		return defaultVal
	}
	return envValBool
}

func getIntArrayFromEnv(envKey string, defaultVal []int) []int {
	var evis []int
	ev, exists := getEnvValByKey(envKey)
	if !exists {
		return defaultVal
	}
	evs := strings.Split(ev, ",")

	for _, ev := range evs {
		evi, err := strconv.Atoi(ev)
		if err != nil {
			// TODO: log warning
			return defaultVal
		}
		evis = append(evis, evi)
	}
	return evis
}

func getIntFromEnv(envKey string, defaultVal int) int {
	ev, exists := getEnvValByKey(envKey)
	if !exists {
		return defaultVal
	}
	evi, err := strconv.Atoi(ev)
	if err != nil {
		return defaultVal
	}
	return evi
}

func getFloat64FromEnv(envKey string, defaultVal float64) float64 {
	ev, exists := getEnvValByKey(envKey)
	if !exists {
		return defaultVal
	}
	evi, err := strconv.ParseFloat(ev, 64)
	if err != nil {
		return defaultVal
	}
	return evi
}

func getStringFromEnv(envKey string, defaultVal string) string {
	ev, exists := getEnvValByKey(envKey)
	if !exists {
		return defaultVal
	}
	return ev
}

func getLogLevelFromEnv(envKey string, defaultVal zerolog.Level) zerolog.Level {
	ev, exists := getEnvValByKey(envKey)
	if !exists {
		return defaultVal
	}
	switch ev {
	case zerolog.DebugLevel.String():
		return zerolog.DebugLevel
	default:
		return defaultVal
	}
}

func getSecTimeDurationFromEnv(envKey string, defaultVal int) time.Duration {
	return time.Duration(getIntFromEnv(envKey, defaultVal)) * time.Second
}

func NewCreateOptions(payload *ChromeConfigOptionsPayload) (co ChromeConfigOptions, err error) {
	hash, err := hashstructure.Hash(&payload, hashstructure.FormatV2, nil)
	if err != nil {
		return co, err
	}
	co.Profile = payload.Profile
	co.Hash = strconv.FormatUint(hash, 10)
	return co, nil
}
