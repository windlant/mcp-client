package skills

import (
	"context"
	"errors"
	"sync"

	"github.com/windlant/mcp-client/internal/config"
	"github.com/windlant/mcp-client/internal/skills/code_driven"
	"github.com/windlant/mcp-client/internal/skills/llm_driven"
	"github.com/windlant/mcp-client/internal/skills/manage/registry"

	"github.com/windlant/protocol/types/agent_types"
	"github.com/windlant/protocol/types/skill_types"
)

// SkillClient 定义技能调用的统一接口
type SkillClient interface {
	Call(ctx context.Context, name string, args skill_types.SkillArguments) (interface{}, error)
	GetSkill(name string) (skill_types.SkillDefinition, bool)
	ListSkills() []string
}

// RemoteSkillMetadata 记录远程 skill 的元数据（来自哪个 agent）
type RemoteSkillMetadata struct {
	SkillDefinition skill_types.SkillDefinition
	SourceAgent     agent_types.AgentCard
}

// RemoteSkillCaller 定义远程 skill 调用的函数类型
type RemoteSkillCaller func(ctx context.Context, targetAgent agent_types.AgentCard, skillID string, input skill_types.SkillArguments) (interface{}, error)

// LLMSkillExecutor 定义 LLM-driven skill 执行的函数类型
// ctx 中可能包含用于执行的 Agent（通过 skills.ContextAgentKey）
type LLMSkillExecutor func(ctx context.Context, skillDef skill_types.SkillDefinition, input skill_types.SkillArguments) (interface{}, error)

// ContextAgentKey 是放在 context 里的 agent 变量名（值为 *agent.Agent）
// 注意：skills 包不直接依赖 agent 类型，context 中的值由调用方传入并在 executor 中断言使用。
type ContextAgentKeyType struct{}

var ContextAgentKey = ContextAgentKeyType{}

// IntegratedClient 集成客户端，拥有合并后的 Registry 视图以及远程 skill 支持
type IntegratedClient struct {
	mergedRegistry  *registry.Registry
	llmClient       *llm_driven.LLMDrivenClient
	codeClient      *code_driven.CodeDrivenClient
	mu              sync.RWMutex
	remoteSkills    map[string]RemoteSkillMetadata // skillID -> RemoteSkillMetadata
	remoteCallFunc  RemoteSkillCaller              // 用于执行远程 skill 调用
	llmExecutorFunc LLMSkillExecutor               // 用于执行本地和远程 LLM-driven skill
}

// NewIntegratedClient 创建集成客户端并完成各自技能注册
func NewIntegratedClient(cfg *config.Config) *IntegratedClient {
	llmClient := llm_driven.NewLLMDrivenClient(cfg)
	codeClient := code_driven.NewCodeDrivenClient()

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
		remoteSkills:   make(map[string]RemoteSkillMetadata),
	}
}

// RegisterRemoteSkillCaller 设置远程 skill 调用的实现函数
func (ic *IntegratedClient) RegisterRemoteSkillCaller(caller RemoteSkillCaller) {
	ic.mu.Lock()
	defer ic.mu.Unlock()
	ic.remoteCallFunc = caller
}

// RegisterLLMSkillExecutor 设置 LLM-driven skill 执行的实现函数
func (ic *IntegratedClient) RegisterLLMSkillExecutor(executor LLMSkillExecutor) {
	ic.mu.Lock()
	defer ic.mu.Unlock()
	ic.llmExecutorFunc = executor
}

// RegisterRemoteAgentSkills 注册来自远程 agent 的 skills
func (ic *IntegratedClient) RegisterRemoteAgentSkills(agent agent_types.AgentCard) {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	for _, skillDef := range agent.Skills {
		skillID := skillDef.Name
		ic.remoteSkills[skillID] = RemoteSkillMetadata{
			SkillDefinition: skillDef,
			SourceAgent:     agent,
		}
	}
}

// UnregisterRemoteAgentSkills 注销来自某个 agent 的全部 skills
func (ic *IntegratedClient) UnregisterRemoteAgentSkills(agentID string) {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	for skillID, metadata := range ic.remoteSkills {
		if metadata.SourceAgent.AgentID == agentID {
			delete(ic.remoteSkills, skillID)
		}
	}
}

// Call 根据技能类型路由到对应的客户端或远程调用
func (ic *IntegratedClient) Call(ctx context.Context, name string, args skill_types.SkillArguments) (interface{}, error) {
	// 首先查询本地 skill
	skill, exists := ic.mergedRegistry.Get(name)
	if exists {
		if skill.Function == nil {
			// LLM-driven skill：使用 executor 执行（或返回 spec 作为 fallback）
			ic.mu.RLock()
			executor := ic.llmExecutorFunc
			ic.mu.RUnlock()

			if executor != nil {
				return executor(ctx, skill, args) // 本地调用，传入 ctx（可能包含 agent）
			}
			// Fallback: 返回 skill.md
			return ic.llmClient.Call(name, args)
		} else {
			// Code-driven skill
			return ic.codeClient.Call(name, args)
		}
	}

	// 查询远程 skill
	ic.mu.RLock()
	remoteMetadata, remoteExists := ic.remoteSkills[name]
	remoteCaller := ic.remoteCallFunc
	ic.mu.RUnlock()

	if !remoteExists {
		return nil, errors.New("skill not found: " + name)
	}

	// 远程 skill 始终通过远程调用转发执行
	if remoteCaller != nil {
		return remoteCaller(ctx, remoteMetadata.SourceAgent, name, args)
	}

	return nil, errors.New("remote skill caller not registered")
}

// GetSkill 获取技能定义（包括远程 skill）
func (ic *IntegratedClient) GetSkill(name string) (skill_types.SkillDefinition, bool) {
	// 优先查询本地 skill
	skill, exists := ic.mergedRegistry.Get(name)
	if exists {
		return skill, true
	}

	// 查询远程 skill
	ic.mu.RLock()
	defer ic.mu.RUnlock()

	remoteMetadata, exists := ic.remoteSkills[name]
	return remoteMetadata.SkillDefinition, exists
}

// ListSkills 列出所有技能（本地 + 远程）
func (ic *IntegratedClient) ListSkills() []string {
	var names []string

	// 添加本地 skills
	for _, skill := range ic.mergedRegistry.ListAll() {
		names = append(names, skill.Name)
	}

	// 添加远程 skills
	ic.mu.RLock()
	for skillID := range ic.remoteSkills {
		names = append(names, skillID)
	}
	ic.mu.RUnlock()

	return names
}

// ListRemoteSkills 列出所有远程 skills（调试用）
func (ic *IntegratedClient) ListRemoteSkills() map[string]RemoteSkillMetadata {
	ic.mu.RLock()
	defer ic.mu.RUnlock()

	result := make(map[string]RemoteSkillMetadata)
	for k, v := range ic.remoteSkills {
		result[k] = v
	}
	return result
}
