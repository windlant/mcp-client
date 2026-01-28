package model

import "github.com/windlant/mcp-client/internal/protocol"

// ToolForAPI represents a tool in the format expected by LLM APIs (e.g., DeepSeek, OpenAI).
type ToolForAPI struct {
	Type     string      `json:"type"` // e.g., "function"
	Function ToolFuncDef `json:"function"`
}

// ToolFuncDef describes a callable function/tool.
type ToolFuncDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"` // JSON Schema object
}

// Model is the unified interface for all LLM backends.
type Model interface {
	// Chat handles standard conversation without tool calling.
	Chat(messages []protocol.Message) (string, error)

	// ChatWithTools handles conversation with tool definitions.
	// The implementation should:
	// - Use 'tools' to guide the model if supported
	// - Return any tool_calls made by the model
	// - For unsupported models, return empty toolCalls and treat as normal chat
	ChatWithTools(messages []protocol.Message, tools []ToolForAPI) (content string, toolCalls []protocol.ToolCall, err error)
}
