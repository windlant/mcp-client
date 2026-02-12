package llm_driven

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/windlant/mcp-client/internal/config"
	"github.com/windlant/mcp-client/internal/skills/manage/registry"

	"github.com/windlant/protocol/types/skill_types"
)

type LLMDrivenClient struct {
	registry *registry.Registry
}

// NewLLMDrivenClient 从配置创建 LLM-driven 客户端并注册所有 LLM-driven skills
func NewLLMDrivenClient(cfg *config.Config) *LLMDrivenClient {
	reg := registry.NewRegistry()

	// 从配置加载并注册 LLM-driven skills
	skillDefs := cfg.ConvertToSkillDefinitions()
	for _, skillDef := range skillDefs {
		// 确保 Function 为 nil（LLM-driven）
		skillDef.Function = nil
		reg.Register(skillDef)
	}

	return &LLMDrivenClient{registry: reg}
}

// Call 对于 LLM-driven skills，返回 SKILL.md 内容让上层处理
func (c *LLMDrivenClient) Call(name string, args skill_types.SkillArguments) (interface{}, error) {
	if _, exists := c.registry.Get(name); !exists {
		return nil, errors.New("skill not found: " + name)
	}

	// 构建 spec 文件路径
	// specs 目录在 internal/skills/manage/builtin/llm_driven_skills/specs/
	specPath := filepath.Join("internal", "skills", "manage", "builtin", "llm_driven_skills", "specs", name+".md")

	// 读取 SKILL.md 内容
	content, err := os.ReadFile(specPath)
	if err != nil {
		// 如果文件不存在，返回有意义的错误信息
		return nil, fmt.Errorf("failed to read skill spec file %s: %w", specPath, err)
	}

	return string(content), nil
}

// GetSkill 获取技能定义
func (c *LLMDrivenClient) GetSkill(name string) (skill_types.SkillDefinition, bool) {
	return c.registry.Get(name)
}

// ListSkills 列出所有 LLM-driven 技能
func (c *LLMDrivenClient) ListSkills() []string {
	var names []string
	for _, skill := range c.registry.ListAll() {
		names = append(names, skill.Name)
	}
	return names
}

// GetRegistry 获取内部注册表（供 IntegratedClient 合并使用）
func (c *LLMDrivenClient) GetRegistry() *registry.Registry {
	return c.registry
}
