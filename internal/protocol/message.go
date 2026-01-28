// internal/protocol/message.go
package protocol

// Message represents a chat message compatible with DeepSeek function calling.
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	Name       string     `json:"name,omitempty"`         // for tool messages
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // assistant -> tool calls
	ToolCallID string     `json:"tool_call_id,omitempty"` // tool -> response to a call
}

// ToolCall represents a single function/tool invocation request from the model.
type ToolCall struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"` // must be "function"
	Function Function `json:"function"`
}

// Function describes the function to be called.
type Function struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON-encoded string, e.g., "{\"location\": \"Beijing\"}"
}
