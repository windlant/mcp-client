package protocol

import (
	"encoding/json"
	"strings"
)

// Message represents a single message in a conversation.
// Compatible with DeepSeek/OpenAI chat format.
type Message struct {
	Role    string `json:"role"`              // "user", "assistant", or "tool"
	Content string `json:"content,omitempty"` // Main text or tool call JSON
	Name    string `json:"name,omitempty"`    // Tool name (required when Role == "tool")
}

// IsToolCall checks if the assistant's message contains a tool call request.
// Expected format: {"tools": [{"name": "...", "arguments": {...}}, ...]}
func (m *Message) IsToolCall() bool {
	if m.Role != "assistant" {
		return false
	}
	content := strings.TrimSpace(m.Content)
	return strings.HasPrefix(content, "{") && strings.Contains(content, `"tools"`)
}

// ExtractToolCalls parses tool calls from the assistant's response content.
// Returns a list of tool calls, or nil if no valid tool calls are found.
// On JSON parse error, returns an error.
func (m *Message) ExtractToolCalls() ([]map[string]interface{}, error) {
	if !m.IsToolCall() {
		return nil, nil
	}

	var wrapper struct {
		Tools []map[string]interface{} `json:"tools"`
	}
	if err := json.Unmarshal([]byte(m.Content), &wrapper); err != nil {
		return nil, err
	}
	return wrapper.Tools, nil
}
