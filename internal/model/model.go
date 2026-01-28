package model

import "github.com/windlant/mcp-client/internal/protocol"

// ToolForAPI 表示 LLM API（如 DeepSeek、OpenAI）所期望的工具格式
type ToolForAPI struct {
	Type     string      `json:"type"` // 例如 "function"
	Function ToolFuncDef `json:"function"`
}

// ToolFuncDef 描述一个可调用的函数/工具
type ToolFuncDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"` // JSON Schema 对象
}

// Model 是所有大语言模型后端的统一接口
type Model interface {
	// Chat 处理不使用工具的标准对话
	Chat(messages []protocol.Message) (string, error)

	// ChatWithTools 处理支持工具调用的对话
	// 实现时应：
	// - 如果模型支持，使用 'tools' 引导模型行为
	// - 返回模型生成的 tool_calls
	// - 对于不支持工具的模型，返回空的 toolCalls，并按普通对话处理
	ChatWithTools(messages []protocol.Message, tools []ToolForAPI) (content string, toolCalls []protocol.ToolCall, err error)
}
