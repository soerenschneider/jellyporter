package config

import (
	"errors"
	"os"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

var validation = validator.New()

const (
	DefaultFullSyncIntervalMinutes = 60 * 6
	DefaultSyncIntervalMinutes     = 5
	DefaultMetricsAddr             = "127.0.0.1:8972"
)

type Config struct {
	Database struct {
		Path string `yaml:"path" validate:"omitempty,filepath"`
	} `yaml:"database"`
	Clients map[string]JellyfinServerConfig `yaml:"clients" validate:"dive"`

	EventSources *Events `yaml:"events"`

	SyncIntervalMinutes     int `yaml:"sync_interval_mins" validate:"gte=5,lt=1440"`
	FullSyncIntervalMinutes int `yaml:"full_sync_interval_mins" validate:"gte=30,lt=1440"`

	MetricsAddr string `yaml:"metrics_addr" validate:"omitempty,hostname_port"`
	MetricsPath string `yaml:"metrics_path" validate:"omitempty,filepath"`
}

type Events struct {
	WebhookServer *struct {
		Addr string `yaml:"addr" validate:"omitempty,hostname_port"`
		Path string `yaml:"path"` // TODO: validate
	} `yaml:"webhook"`
}

type JellyfinServerConfig struct {
	Address    string `yaml:"url" validate:"http_url"`
	User       string `yaml:"user" validate:"alphanum"`
	ApiKey     string `yaml:"api_key" validate:"required_without=ApiKeyFile,omitempty,alphanum"`
	ApiKeyFile string `yaml:"api_key_file" validate:"required_without=ApiKey,omitempty,file"`
}

func (c *JellyfinServerConfig) GetApiKey() (string, error) {
	if c.ApiKey != "" {
		return c.ApiKey, nil
	}

	data, err := os.ReadFile(c.ApiKeyFile)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if err := validation.Struct(c); err != nil {
		return err
	}

	if c.FullSyncIntervalMinutes%c.SyncIntervalMinutes != 0 {
		return errors.New("full_sync_interval_mins must be divisible by sync_interval_mins but is not")
	}

	return nil
}

func (c *Config) UnmarshalYAML(node *yaml.Node) error {
	type Alias Config // Create an alias to avoid recursion during unmarshalling

	// Define a temporary struct with default values
	tmp := &Alias{
		FullSyncIntervalMinutes: DefaultFullSyncIntervalMinutes,
		SyncIntervalMinutes:     DefaultSyncIntervalMinutes,
		MetricsAddr:             DefaultMetricsAddr,
	}

	// Unmarshal the yaml data into the temporary struct
	if err := node.Decode(&tmp); err != nil {
		return err
	}

	// Assign the values from the temporary struct to the original struct
	*c = Config(*tmp)
	return nil
}
