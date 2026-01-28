package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	APIKey  string        `yaml:"api_key"`
	Model   ModelConfig   `yaml:"model"`
	Context ContextConfig `yaml:"context"`
	Tools   ToolsConfig   `yaml:"tools"`
}

type ModelConfig struct {
	Provider    string  `yaml:"provider"`
	ModelName   string  `yaml:"model_name"`
	Temperature float32 `yaml:"temperature"`
	MaxTokens   int     `yaml:"max_tokens"`
}

type ContextConfig struct {
	MaxHistory int `yaml:"max_history"`
}

type ToolsConfig struct {
	Enabled bool `yaml:"enabled"`
}

// Load 从 config/config.yaml 加载配置
func Load() (*Config, error) {
	data, err := os.ReadFile("config/config.yaml")
	if err != nil {
		return nil, err
	}
	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	return &cfg, err
}
