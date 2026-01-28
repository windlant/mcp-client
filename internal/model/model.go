package model

import "github.com/windlant/mcp-client/internal/protocol"

// Model defines the interface for any LLM backend.
type Model interface {
	// Chat sends a list of messages and returns the model's response text.
	// The response may be a natural language reply or a tool call in JSON format.
	Chat(messages []protocol.Message) (string, error)
}
