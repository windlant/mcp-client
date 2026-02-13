package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/windlant/mcp-client/internal/model"
	"github.com/windlant/mcp-client/internal/skills"
	"github.com/windlant/mcp-client/internal/tools"
	"github.com/windlant/protocol/protocol/llm_protocol"
	"github.com/windlant/protocol/types/skill_types"
	"github.com/windlant/protocol/types/tools_types"
)

// Agent 是智能对话代理，负责管理对话历史、调用模型和工具
type Agent struct {
	model        model.Model                 // 使用的语言模型
	toolClient   *tools.AggregatedToolClient // 工具客户端（用于调用外部功能）
	skillClient  skills.SkillClient          // 技能客户端（用于调用本地技能）
	history      []llm_protocol.Message      // 对话历史记录
	maxMessages  int                         // 最大保存的历史消息数（不含 system 消息）
	toolsEnabled bool                        // 是否启用工具调用功能

	// 内部缓存用于工具分类
	localToolDefs     map[string]tools_types.ToolDefinition // 本地工具定义
	allOtherToolNames []string                              // 其他所有工具/技能名称（用于系统提示）
}

// NewAgent 创建一个新的智能代理
func NewAgent(m model.Model, maxHistory int, toolsEnabled bool, toolClient *tools.AggregatedToolClient, skillClient skills.SkillClient) *Agent {
	if maxHistory <= 0 {
		maxHistory = 20 // 默认最多保留 20 条消息
	}

	agent := &Agent{
		model:         m,
		toolClient:    toolClient,
		skillClient:   skillClient,
		history:       make([]llm_protocol.Message, 0),
		maxMessages:   maxHistory,
		toolsEnabled:  toolsEnabled,
		localToolDefs: make(map[string]tools_types.ToolDefinition),
	}

	// 初始化工具分类（如果启用了工具）
	if toolsEnabled && toolClient != nil {
		agent.initializeToolClassification()
	}

	return agent
}

// initializeToolClassification 初始化并分类工具
func (a *Agent) initializeToolClassification() {
	// 获取所有工具
	allToolDefs, err := a.toolClient.List()
	if err != nil {
		fmt.Printf("⚠ 警告: 无法获取工具列表: %v\n", err)
		return
	}

	// 分类工具：只将 local 工具放入可调用列表
	localTools := make(map[string]tools_types.ToolDefinition)
	var otherToolNames []string

	for _, toolDef := range allToolDefs {
		source, exists := a.toolClient.GetToolSource(toolDef.Name)
		if exists && source == tools.ToolSourceLocal {
			localTools[toolDef.Name] = toolDef
		} else {
			// remote 或 stdio 工具，只记录名称
			otherToolNames = append(otherToolNames, toolDef.Name)
		}
	}

	// 添加技能名称到其他工具列表
	skillNames := a.skillClient.ListSkills()
	otherToolNames = append(otherToolNames, skillNames...)

	a.localToolDefs = localTools
	a.allOtherToolNames = otherToolNames
}

// getSystemPrompt 生成系统提示词，明确指导如何使用非本地工具
func (a *Agent) getSystemPrompt() string {
	basePrompt := "You are a helpful, precise, and proactive AI assistant."

	if a.toolsEnabled && len(a.allOtherToolNames) > 0 {
		toolList := strings.Join(a.allOtherToolNames, ", ")
		extendedPrompt := fmt.Sprintf(`%s

## Advanced Capabilities Notice

You have access to additional advanced tools and skills beyond the directly callable ones. Their names are:
%s

⚠️ IMPORTANT: These tools are NOT directly callable. If you determine that one of them is needed to fulfill the user's request:

1. FIRST, call the **get_tool_detail** tool with the exact name of the desired tool/skill.
2. Carefully examine the returned definition (including parameters, description, and usage rules).
3. ONLY THEN, if the tool is appropriate and all required parameters can be determined, proceed to use it by calling it explicitly in a subsequent step.

NEVER attempt to call these advanced tools directly without first retrieving their full specification via get_tool_detail.

Your goal is to solve the user's task reliably and safely. When in doubt, use get_tool_detail to clarify.`, basePrompt, toolList)
		return extendedPrompt
	}

	return basePrompt
}

// trimHistory 修剪对话历史，确保不超过最大消息数（system 消息除外）
func (a *Agent) trimHistory() {
	if len(a.history) == 0 {
		return
	}

	// 先找 system 消息的位置
	systemIdx := -1
	for i, msg := range a.history {
		if msg.Role == "system" {
			systemIdx = i
			break
		}
	}

	// 分离出非 system 的消息
	nonSystemMsgs := a.history
	if systemIdx >= 0 {
		nonSystemMsgs = a.history[systemIdx+1:]
	}

	// 如果非 system 消息太多，就截断
	if len(nonSystemMsgs) > a.maxMessages {
		keepStart := len(nonSystemMsgs) - a.maxMessages
		trimmed := nonSystemMsgs[keepStart:]

		// 重新组合：保留 system + 最新的消息
		if systemIdx >= 0 {
			a.history = append([]llm_protocol.Message{a.history[systemIdx]}, trimmed...)
		} else {
			a.history = trimmed
		}
	}
}

// Chat 处理用户输入并返回助手的回复
// 支持多轮工具调用（最多 10 轮）
func (a *Agent) Chat(input string) (string, error) {
	return a.ChatWithContext(context.Background(), input)
}

// ChatWithContext 与 Chat 等价，但允许调用方传入 base context（用于传递绑定的 Agent 等）
func (a *Agent) ChatWithContext(baseCtx context.Context, input string) (string, error) {
	// 如果是第一次对话，添加 system 提示
	if len(a.history) == 0 {
		systemMsg := llm_protocol.Message{
			Role:    "system",
			Content: a.getSystemPrompt(),
		}
		a.history = append(a.history, systemMsg)
	}

	// 添加用户消息
	a.history = append(a.history, llm_protocol.Message{
		Role:    "user",
		Content: input,
	})
	a.trimHistory()

	// 获取工具定义（如果启用了工具）- 只包含本地工具 + get_tool_detail
	var apiTools []model.ToolForAPI
	if a.toolsEnabled {
		apiTools = a.getLocalToolsForAPI()
	}

	maxRounds := 10
	for round := 0; round < maxRounds; round++ {
		var content string
		var toolCalls []llm_protocol.ToolCall
		var err error

		// 调用模型，可能返回文本内容或工具调用请求
		content, toolCalls, err = a.model.ChatWithTools(a.history, apiTools)
		if err != nil {
			return "", fmt.Errorf("failed to call model: %w\n", err)
		}

		// 构造助手的回复消息（可能包含工具调用）
		assistantMsg := llm_protocol.Message{
			Role:      "assistant",
			Content:   content,
			ToolCalls: toolCalls,
		}
		a.history = append(a.history, assistantMsg)
		a.trimHistory()

		// 如果没有工具调用，直接返回最终答案
		if len(toolCalls) == 0 {
			return content, nil
		}
		log.Printf("模型请求调用工具: %v", toolCalls)
		// 执行每个工具调用
		for _, tc := range toolCalls {
			// 解析工具参数（JSON 字符串转为 map）
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				// 参数解析失败，记录错误
				a.history = append(a.history, llm_protocol.Message{
					Role:       "tool",
					Name:       tc.Function.Name,
					ToolCallID: tc.ID,
					Content:    "Error: invalid arguments JSON",
				})
				continue
			}

			// 特殊处理 get_tool_detail 工具
			var result interface{}
			var callErr error

			if tc.Function.Name == "get_tool_detail" {
				result, callErr = a.handleGetToolDetail(args)
			} else {
				// 尝试调用 Skill（优先）
				if _, isSkill := a.skillClient.GetSkill(tc.Function.Name); isSkill {
					// 将当前 Agent 放入 context 以便 executor 获取并保持/隔离上下文
					ctx := baseCtx
					if ctx == nil {
						ctx = context.Background()
					}
					ctx = context.WithValue(ctx, skills.ContextAgentKey, a)
					result, callErr = a.skillClient.Call(ctx, tc.Function.Name, skill_types.SkillArguments(args))
				} else {
					// 调用 Tool（通过聚合客户端）
					result, callErr = a.toolClient.Call(tc.Function.Name, args)
				}
			}

			if callErr != nil {
				// 执行失败，记录错误
				a.history = append(a.history, llm_protocol.Message{
					Role:       "tool",
					Name:       tc.Function.Name,
					ToolCallID: tc.ID,
					Content:    "Error: " + callErr.Error(),
				})
				continue
			}

			// 成功执行，记录结果（转换为字符串）
			resultStr := ""
			if result != nil {
				switch v := result.(type) {
				case string:
					resultStr = v
				default:
					// 尝试 JSON 序列化
					if jsonBytes, err := json.Marshal(result); err == nil {
						resultStr = string(jsonBytes)
					} else {
						resultStr = fmt.Sprintf("%v", result)
					}
				}
			}

			a.history = append(a.history, llm_protocol.Message{
				Role:       "tool",
				Name:       tc.Function.Name,
				ToolCallID: tc.ID,
				Content:    resultStr,
			})
		}
		a.trimHistory()
	}

	// 超过最大轮数仍未完成，返回错误提示
	last := a.history[len(a.history)-1]
	if last.Role == "assistant" && len(last.ToolCalls) > 0 {
		return "Error: Maximum tool call depth exceeded.", nil
	}
	return last.Content, nil
}

// handleGetToolDetail 处理 get_tool_detail 工具调用
func (a *Agent) handleGetToolDetail(args map[string]interface{}) (string, error) {
	name, exists := args["name"]
	if !exists {
		return "", fmt.Errorf("missing required parameter 'name'")
	}

	nameStr, ok := name.(string)
	if !ok {
		return "", fmt.Errorf("parameter 'name' must be a string")
	}

	// 首先检查是否是本地工具
	if toolDef, exists := a.localToolDefs[nameStr]; exists {
		defJson, err := json.MarshalIndent(toolDef, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal tool definition: %w", err)
		}
		return string(defJson), nil
	}

	// 然后检查是否是其他工具（remote/stdio）
	if a.toolClient != nil {
		allTools, err := a.toolClient.List()
		if err == nil {
			for _, toolDef := range allTools {
				if toolDef.Name == nameStr {
					defJson, err := json.MarshalIndent(toolDef, "", "  ")
					if err != nil {
						return "", fmt.Errorf("failed to marshal tool definition: %w", err)
					}
					return string(defJson), nil
				}
			}
		}
	}

	// 最后检查是否是技能
	if skillDef, exists := a.skillClient.GetSkill(nameStr); exists {
		defJson, err := json.MarshalIndent(skillDef, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal skill definition: %w", err)
		}
		return string(defJson), nil
	}

	return "", fmt.Errorf("tool or skill '%s' not found", nameStr)
}

// getLocalToolsForAPI 只返回本地工具用于 API 调用（包括 get_tool_detail）
func (a *Agent) getLocalToolsForAPI() []model.ToolForAPI {
	var localTools []model.ToolForAPI

	// 添加现有的本地工具
	for _, toolDef := range a.localToolDefs {
		localTools = append(localTools, convertToolDefsToAPI([]tools_types.ToolDefinition{toolDef})...)
	}

	// 添加 get_tool_detail 工具（硬编码）
	getToolDetailDef := tools_types.ToolDefinition{
		Name:        "get_tool_detail",
		Description: "Get the detailed definition (name, description, parameters schema) of a specific tool or skill by its name.",
		Parameters: tools_types.ToolSchema{
			Type: "object",
			Properties: map[string]tools_types.ToolParameter{
				"name": {
					Type:        "string",
					Description: "The exact name of the tool or skill to get details for",
				},
			},
			Required: []string{"name"},
		},
		Function: nil, // 这个工具由 Agent 特殊处理
	}
	localTools = append(localTools, convertToolDefsToAPI([]tools_types.ToolDefinition{getToolDetailDef})...)

	return localTools
}

// ClearHistory 清空对话历史（重置上下文）
func (a *Agent) ClearHistory() {
	a.history = make([]llm_protocol.Message, 0)
}

// convertToolDefsToAPI 将内部工具定义转换为模型 API 所需的格式
func convertToolDefsToAPI(defs []tools_types.ToolDefinition) []model.ToolForAPI {
	apiTools := make([]model.ToolForAPI, len(defs))
	for i, def := range defs {
		// 构建参数属性
		props := make(map[string]interface{})
		for name, param := range def.Parameters.Properties {
			props[name] = map[string]interface{}{
				"type":        param.Type,
				"description": param.Description,
			}
		}

		// 构建完整的 JSON Schema
		schema := map[string]interface{}{
			"type":       "object",
			"properties": props,
		}
		if len(def.Parameters.Required) > 0 {
			schema["required"] = def.Parameters.Required
		}

		// 转换为模型所需的工具格式
		apiTools[i] = model.ToolForAPI{
			Type: "function",
			Function: model.ToolFuncDef{
				Name:        def.Name,
				Description: def.Description,
				Parameters:  schema,
			},
		}
	}
	return apiTools
}
