package config

import (
	"fmt"
	"os"

	"github.com/windlant/protocol/types/skill_types"
	"gopkg.in/yaml.v3"
)

// Config 表示完整的应用程序配置
type Config struct {
	Model   ModelConfig   `yaml:"model"`
	Context ContextConfig `yaml:"context"`
	Tools   ToolsConfig   `yaml:"tools"`
	Agent   AgentConfig   `yaml:"agent"`
}

// ModelConfig 表示大语言模型设置
type ModelConfig struct {
	APIKey      string  `yaml:"api_key"`
	Provider    string  `yaml:"provider"`
	ModelName   string  `yaml:"model_name"`
	Temperature float32 `yaml:"temperature"`
	MaxTokens   int     `yaml:"max_tokens"`
}

// ContextConfig 表示对话上下文管理设置
type ContextConfig struct {
	MaxHistory int `yaml:"max_history"`
}

// ToolsConfig 表示 MCP 工具客户端设置
type ToolsConfig struct {
	Enabled   bool            `yaml:"enabled"`    // 是否启用工具功能
	Mode      string          `yaml:"mode"`       // 工具模式（保留字段）
	URL       string          `yaml:"url"`        // 工具服务 URL（可选）
	MCPServer MCPServerConfig `yaml:"mcp_server"` // MCP 服务器配置
}

// MCPServerConfig 表示 MCP 服务器连接设置
type MCPServerConfig struct {
	Host string `yaml:"host"` // MCP 服务器主机地址
	Port int    `yaml:"port"` // MCP 服务器端口
}

// AgentConfig 表示 Agent 相关设置
type AgentConfig struct {
	// Agent 基本信息（用于 A2A AgentCard）
	ID          string       `yaml:"id"`          // Agent 唯一标识符
	Name        string       `yaml:"name"`        // Agent 名称
	Description string       `yaml:"description"` // Agent 描述
	A2A         A2AConfig    `yaml:"a2a"`         // A2A 协议相关设置
	Skills      SkillsConfig `yaml:"skills"`
}

// SkillsConfig 表示 Agent 的技能配置
type SkillsConfig struct {
	LLMDriven []LLMDrivenSkillConfig `yaml:"llm_driven"`
}

// LLMDrivenSkillConfig 表示 LLM-driven Skill 的配置
type LLMDrivenSkillConfig struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Parameters  SkillSchemaConfig `yaml:"parameters"`
}

// SkillSchemaConfig 对应 skill_types.SkillSchema
type SkillSchemaConfig struct {
	Type       string                      `yaml:"type"`
	Properties map[string]SkillParamConfig `yaml:"properties"`
	Required   []string                    `yaml:"required"`
}

// SkillParamConfig 对应 skill_types.SkillParameter
type SkillParamConfig struct {
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"` // 注意：这个字段主要用于文档，在 Required 列表中才是真正的必需标识
}

// A2AConfig 表示 A2A 协议相关设置
type A2AConfig struct {
	Enabled       bool   `yaml:"enabled"`         // 是否启用 A2A 功能
	Host          string `yaml:"host"`            // A2A 服务监听主机
	Port          int    `yaml:"port"`            // A2A 服务监听端口
	WellKnownPath string `yaml:"well_known_path"` // AgentCard 路径
}

// Load 从 YAML 文件加载配置
func Load() (*Config, error) {
	// 读取配置文件
	data, err := os.ReadFile("config/config.yaml")
	if err != nil {
		return nil, err
	}

	// 解析 YAML 配置
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// 为上下文配置应用默认值
	if cfg.Context.MaxHistory <= 0 {
		cfg.Context.MaxHistory = 20
	}

	// 为模型配置应用默认值
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

	// 为 Agent 配置应用默认值
	if cfg.Agent.ID == "" {
		cfg.Agent.ID = "my-agent-v1"
	}
	if cfg.Agent.Name == "" {
		cfg.Agent.Name = "My Personal Assistant"
	}
	if cfg.Agent.Description == "" {
		cfg.Agent.Description = "A personal assistant that can help with various tasks using both LLM-driven and code-driven skills."
	}

	// 为 A2A 配置应用默认值
	if !cfg.Agent.A2A.Enabled {
		// 如果 A2A 未显式启用，仍然设置默认值以便将来使用
		if cfg.Agent.A2A.Host == "" {
			cfg.Agent.A2A.Host = "0.0.0.0"
		}
		if cfg.Agent.A2A.Port == 0 {
			cfg.Agent.A2A.Port = 8080
		}
		if cfg.Agent.A2A.WellKnownPath == "" {
			cfg.Agent.A2A.WellKnownPath = "/.well-known/agent.json"
		}
	} else {
		// 如果 A2A 已启用，确保必要字段已设置
		if cfg.Agent.A2A.Host == "" {
			cfg.Agent.A2A.Host = "0.0.0.0"
		}
		if cfg.Agent.A2A.Port == 0 {
			cfg.Agent.A2A.Port = 8080
		}
		if cfg.Agent.A2A.WellKnownPath == "" {
			cfg.Agent.A2A.WellKnownPath = "/.well-known/agent.json"
		}
	}

	// 验证必需的配置字段
	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("配置无效: %w", err)
	}

	return &cfg, nil
}

// validateConfig 验证必需的配置字段是否已提供
func validateConfig(cfg *Config) error {
	// 验证模型配置
	if cfg.Model.APIKey == "" {
		return fmt.Errorf("model.api_key 是必需的")
	}

	// 验证 Agent 配置
	if cfg.Agent.ID == "" {
		return fmt.Errorf("agent.id 是必需的")
	}
	if cfg.Agent.Name == "" {
		return fmt.Errorf("agent.name 是必需的")
	}
	if cfg.Agent.Description == "" {
		return fmt.Errorf("agent.description 是必需的")
	}

	// 验证工具配置（如果已启用）
	if cfg.Tools.Enabled {
		if cfg.Tools.MCPServer.Host == "" && cfg.Tools.URL == "" {
			return fmt.Errorf("当启用工具功能时，必须提供 tools.mcp_server.host 或 tools.url")
		}
		if cfg.Tools.MCPServer.Port <= 0 && cfg.Tools.URL == "" {
			return fmt.Errorf("当启用工具功能时，必须提供 tools.mcp_server.port 或 tools.url")
		}
	}

	for i, skill := range cfg.Agent.Skills.LLMDriven {
		if skill.Name == "" {
			return fmt.Errorf("agent.skills.llm_driven[%d].name 是必需的", i)
		}
		if skill.Description == "" {
			return fmt.Errorf("agent.skills.llm_driven[%d].description 是必需的", i)
		}
		if skill.Parameters.Type == "" {
			return fmt.Errorf("agent.skills.llm_driven[%d].parameters.type 是必需的", i)
		}
	}
	return nil
}

// ConvertToSkillDefinitions 将配置中的技能转换为 skill_types.SkillDefinition 列表
func (cfg *Config) ConvertToSkillDefinitions() []skill_types.SkillDefinition {
	var skills []skill_types.SkillDefinition

	for _, skillCfg := range cfg.Agent.Skills.LLMDriven {
		// 转换参数 schema
		properties := make(map[string]skill_types.SkillParameter)
		for paramName, paramCfg := range skillCfg.Parameters.Properties {
			properties[paramName] = skill_types.SkillParameter{
				Type:        paramCfg.Type,
				Description: paramCfg.Description,
				Required:    paramCfg.Required,
			}
		}

		skillSchema := skill_types.SkillSchema{
			Type:       skillCfg.Parameters.Type,
			Properties: properties,
			Required:   skillCfg.Parameters.Required,
		}

		skillDef := skill_types.SkillDefinition{
			Name:        skillCfg.Name,
			Description: skillCfg.Description,
			Parameters:  skillSchema,
			Function:    nil, // LLM-driven skills have no function implementation
		}

		skills = append(skills, skillDef)
	}

	return skills
}
