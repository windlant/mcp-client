package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Model   ModelConfig   `yaml:"model"`
	Context ContextConfig `yaml:"context"`
	Tools   ToolsConfig   `yaml:"tools"`
}

type ModelConfig struct {
	APIKey      string  `yaml:"api_key"`
	Provider    string  `yaml:"provider"`
	ModelName   string  `yaml:"model_name"`
	Temperature float32 `yaml:"temperature"`
	MaxTokens   int     `yaml:"max_tokens"`
}

type ContextConfig struct {
	MaxHistory int `yaml:"max_history"`
}

type ToolsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Mode    string `yaml:"mode"`
}

func Load() (*Config, error) {
	data, err := os.ReadFile("config/config.yaml")
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.Context.MaxHistory <= 0 {
		cfg.Context.MaxHistory = 20
	}
	if cfg.Model.Temperature == 0 {
		cfg.Model.Temperature = 0.7
	}
	if cfg.Model.MaxTokens == 0 {
		cfg.Model.MaxTokens = 1024
	}
	if cfg.Model.Provider == "" {
		cfg.Model.Provider = "deepseek"
	}
	if cfg.Model.ModelName == "" {
		cfg.Model.ModelName = "deepseek-chat"
	}

	return &cfg, nil
}
