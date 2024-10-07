package cmd

import (
	"fmt"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
)

// Global koanf instance. Use . as the key path delimiter. This can be / or anything.
var (
	k      = koanf.New(".")
	config *Config
)

type Config struct {
	Logger struct {
		Level    string `yaml:"port"`
		Encoding string `yaml:"encoding"`
		Color    bool   `yaml:"color"`
		Output   string `yaml:"output"`
	} `yaml:"logger"`

	Metrics struct {
		Enabled bool   `yaml:"enabled"`
		Host    string `yaml:"host"`
		Port    int    `yaml:"port"`
	} `yaml:"metrics"`

	Profiler struct {
		Enabled bool   `yaml:"enabled"`
		Pidfile string `yaml:"pidfile"`
	} `yaml:"profiler"`

	Server struct {
		Port     int    `yaml:"enabled"`
		Tls      bool   `yaml:"tls"`
		Devcert  bool   `yaml:"devcert"`
		Certfile string `yaml:"certfile"`
		Keyfile  string `yaml:"keyfile"`

		Log struct {
			Enabled      bool     `yaml:"enabled"`
			Level        string   `yaml:"level"`
			RequestBody  bool     `yaml:"request_body"`
			ResponseBody bool     `yaml:"response_body"`
			IgnorePaths  []string `yaml:"ignore_paths"`
		} `yaml:"log"`

		CORS struct {
			Enabled          bool     `yaml:"enabled"`
			AllowedOrigins   []string `yaml:"allowed_origins"`
			AllowedMethods   []string `yaml:"allowed_methods"`
			AllowedHeaders   []string `yaml:"allowed_headers"`
			AllowCredentials bool     `yaml:"allow_credentials"`
			MaxAge           int      `yaml:"max_age"`
		} `yaml:"cors"`

		Metrics struct {
			Enabled     bool     `yaml:"enabled"`
			IgnorePaths []string `yaml:"ignore_paths"`
		} `yaml:"metrics"`

		Database struct {
			Username            string `yaml:"username"`
			Password            string `yaml:"password"`
			Host                string `yaml:"host"`
			Port                int    `yaml:"port"`
			Database            string `yaml:"database"`
			AutoCreate          bool   `yaml:"auto_create"`
			Schema              string `yaml:"schema"`
			SearchPath          string `yaml:"search_path"`
			SSLMode             string `yaml:"sslmode"`
			SSLCert             string `yaml:"sslcert"`
			SSLKey              string `yaml:"sslkey"`
			SSLRootCert         string `yaml:"sslrootcert"`
			Retries             int    `yaml:"retries"`
			SleepBetweenRetries string `yaml:"sleep_between_retries"` // Can be parsed as duration
			MaxConnections      int    `yaml:"max_connections"`
			LogQueries          bool   `yaml:"log_queries"`
			WipeConfirm         bool   `yaml:"wipe_confirm"`
		} `yaml:"database"`
	} `yaml:"server"`
}

func Load() (*Config, error) {
	if config != nil {
		return config, nil
	}

	k.Load(file.Provider("cmd/defaults.yaml"), yaml.Parser())
	if k.Raw() == nil || len(k.Raw()) == 0 {
		return nil, fmt.Errorf("could not load config: %s", "defaults.yaml")
	}

	if err := k.Unmarshal("", &config); err != nil {
		return nil, err
	}

	return config, nil
}
