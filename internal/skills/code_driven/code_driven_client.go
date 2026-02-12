package code_driven

import (
	"errors"

	"github.com/windlant/mcp-client/internal/skills/manage/registry"

	"github.com/windlant/protocol/types/skill_types"
)

type CodeDrivenClient struct {
	registry *registry.Registry
}

// NewCodeDrivenClient 创建 Code-driven 客户端（当前不注册任何技能，预留扩展）
func NewCodeDrivenClient() *CodeDrivenClient {
	reg := registry.NewRegistry()

	// TODO: 未来在这里注册内置的 Code-driven skills
	// 例如：reg.Register(builtInCalculateSkill)
	//      reg.Register(builtInFileReadSkill)

	return &CodeDrivenClient{registry: reg}
}

// RegisterSkill 注册单个 Code-driven skill（供未来动态注册使用）
func (c *CodeDrivenClient) RegisterSkill(def skill_types.SkillDefinition) {
	// 确保是 Code-driven skill（Function 不为 nil）
	if def.Function == nil {
		// 可以选择 panic 或忽略，这里选择忽略并记录日志（简化版直接注册）
		// 实际项目中可能需要更严格的验证
	}
	c.registry.Register(def)
}

// Call 执行 Code-driven skill 的实际函数实现
func (c *CodeDrivenClient) Call(name string, args skill_types.SkillArguments) (interface{}, error) {
	skillDef, exists := c.registry.Get(name)
	if !exists {
		return nil, errors.New("skill not found: " + name)
	}

	if skillDef.Function == nil {
		return nil, errors.New("skill is not code-driven: " + name)
	}

	return skillDef.Function(args)
}

// GetSkill 获取技能定义
func (c *CodeDrivenClient) GetSkill(name string) (skill_types.SkillDefinition, bool) {
	return c.registry.Get(name)
}

// ListSkills 列出所有 Code-driven 技能
func (c *CodeDrivenClient) ListSkills() []string {
	var names []string
	for _, skill := range c.registry.ListAll() {
		names = append(names, skill.Name)
	}
	return names
}

// GetRegistry 获取内部注册表（供 IntegratedClient 合并使用）
func (c *CodeDrivenClient) GetRegistry() *registry.Registry {
	return c.registry
}
