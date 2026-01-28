package agent

import (
	"encoding/json"
	"fmt"

	"github.com/windlant/mcp-client/internal/model"
	"github.com/windlant/mcp-client/internal/protocol"
	"github.com/windlant/mcp-client/internal/tools"
)

// Agent 是智能对话代理，负责管理对话历史、调用模型和工具
type Agent struct {
	model        model.Model        // 使用的语言模型
	toolClient   tools.ToolClient   // 工具客户端（用于调用外部功能）
	history      []protocol.Message // 对话历史记录
	maxMessages  int                // 最大保存的历史消息数（不含 system 消息）
	toolsEnabled bool               // 是否启用工具调用功能
}

// NewAgent 创建一个新的智能代理
// 如果禁用了工具，toolClient 可以为 nil，内部会自动使用空操作客户端
func NewAgent(m model.Model, maxHistory int, toolsEnabled bool, toolClient tools.ToolClient) *Agent {
	if maxHistory <= 0 {
		maxHistory = 20 // 默认最多保留 20 条消息
	}
	if toolClient == nil {
		toolClient = &tools.NoopToolClient{} // 使用空工具客户端避免空指针
	}
	return &Agent{
		model:        m,
		toolClient:   toolClient,
		history:      make([]protocol.Message, 0),
		maxMessages:  maxHistory,
		toolsEnabled: toolsEnabled,
	}
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
			a.history = append([]protocol.Message{a.history[systemIdx]}, trimmed...)
		} else {
			a.history = trimmed
		}
	}
}

// Chat 处理用户输入并返回助手的回复
// 支持多轮工具调用（最多 3 轮）
func (a *Agent) Chat(input string) (string, error) {
	// 如果是第一次对话，添加 system 提示
	if len(a.history) == 0 {
		systemMsg := protocol.Message{
			Role:    "system",
			Content: "You are a helpful assistant.",
		}
		a.history = append(a.history, systemMsg)
	}

	// 添加用户消息
	a.history = append(a.history, protocol.Message{
		Role:    "user",
		Content: input,
	})
	a.trimHistory()

	// 获取工具定义（如果启用了工具）
	var apiTools []model.ToolForAPI
	if a.toolsEnabled {
		defs, err := a.toolClient.List()
		if err != nil {
			return "", fmt.Errorf("failed to list tools: %w", err)
		} else {
			apiTools = convertToolDefsToAPI(defs)
		}
	}

	// 最多进行 5 轮工具调用（防止无限循环）
	maxRounds := 5
	for round := 0; round < maxRounds; round++ {
		var content string
		var toolCalls []protocol.ToolCall
		var err error

		// 调用模型，可能返回文本内容或工具调用请求
		content, toolCalls, err = a.model.ChatWithTools(a.history, apiTools)
		if err != nil {
			return "", fmt.Errorf("failed to call model: %w", err)
		}

		// 构造助手的回复消息（可能包含工具调用）
		assistantMsg := protocol.Message{
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

		// 执行每个工具调用
		for _, tc := range toolCalls {
			// 解析工具参数（JSON 字符串转为 map）
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				// 参数解析失败，记录错误
				a.history = append(a.history, protocol.Message{
					Role:       "tool",
					Name:       tc.Function.Name,
					ToolCallID: tc.ID,
					Content:    "Error: invalid arguments JSON",
				})
				continue
			}

			// 调用工具
			result, err := a.toolClient.Call(tc.Function.Name, args)
			if err != nil {
				// 工具执行失败，记录错误
				a.history = append(a.history, protocol.Message{
					Role:       "tool",
					Name:       tc.Function.Name,
					ToolCallID: tc.ID,
					Content:    "Error: " + err.Error(),
				})
				continue
			}

			// 工具成功执行，记录结果
			a.history = append(a.history, protocol.Message{
				Role:       "tool",
				Name:       tc.Function.Name,
				ToolCallID: tc.ID,
				Content:    result,
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

// ClearHistory 清空对话历史（重置上下文）
func (a *Agent) ClearHistory() {
	a.history = make([]protocol.Message, 0)
}

// convertToolDefsToAPI 将内部工具定义转换为模型 API 所需的格式
func convertToolDefsToAPI(defs []tools.ToolDefinition) []model.ToolForAPI {
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
