package skills

import (
	"errors"

	"github.com/windlant/mcp-client/internal/config"
	"github.com/windlant/mcp-client/internal/skills/code_driven"
	"github.com/windlant/mcp-client/internal/skills/llm_driven"
	"github.com/windlant/mcp-client/internal/skills/manage/registry"

	"github.com/windlant/protocol/types/skill_types"
)

// SkillClient 定义技能调用的统一接口
type SkillClient interface {
	Call(name string, args skill_types.SkillArguments) (interface{}, error)
	GetSkill(name string) (skill_types.SkillDefinition, bool)
	ListSkills() []string
}

// IntegratedClient 集成客户端，拥有合并后的 Registry 视图
type IntegratedClient struct {
	mergedRegistry *registry.Registry
	llmClient      *llm_driven.LLMDrivenClient
	codeClient     *code_driven.CodeDrivenClient
}

// NewIntegratedClient 创建集成客户端并完成各自技能注册
func NewIntegratedClient(cfg *config.Config) *IntegratedClient {
	// 1. 创建各自的客户端（自动完成技能注册）
	llmClient := llm_driven.NewLLMDrivenClient(cfg)
	codeClient := code_driven.NewCodeDrivenClient()

	// 2. 合并 Registry
	mergedReg := registry.NewRegistry()

	// 添加 LLM-driven skills
	for _, skill := range llmClient.GetRegistry().ListAll() {
		mergedReg.Register(skill)
	}

	// 添加 Code-driven skills
	for _, skill := range codeClient.GetRegistry().ListAll() {
		mergedReg.Register(skill)
	}

	return &IntegratedClient{
		mergedRegistry: mergedReg,
		llmClient:      llmClient,
		codeClient:     codeClient,
	}
}

// Call 根据技能类型路由到对应的客户端
func (ic *IntegratedClient) Call(name string, args skill_types.SkillArguments) (interface{}, error) {
	skill, exists := ic.mergedRegistry.Get(name)
	if !exists {
		return nil, errors.New("skill not found: " + name)
	}

	if skill.Function == nil {
		// LLM-driven skill
		return ic.llmClient.Call(name, args)
	} else {
		// Code-driven skill
		return ic.codeClient.Call(name, args)
	}
}

// GetSkill 获取技能定义
func (ic *IntegratedClient) GetSkill(name string) (skill_types.SkillDefinition, bool) {
	return ic.mergedRegistry.Get(name)
}

// ListSkills 列出所有技能
func (ic *IntegratedClient) ListSkills() []string {
	var names []string
	for _, skill := range ic.mergedRegistry.ListAll() {
		names = append(names, skill.Name)
	}
	return names
}
