// internal/agent/agent.go
package agent

import (
	"encoding/json"
	"fmt"

	"github.com/windlant/mcp-client/internal/model"
	"github.com/windlant/mcp-client/internal/protocol"
	"github.com/windlant/mcp-client/internal/tools"
)

type Agent struct {
	model        model.Model
	toolClient   tools.ToolClient
	history      []protocol.Message
	maxMessages  int
	toolsEnabled bool
}

// NewAgent creates a new agent.
// toolClient can be nil if tools are disabled; a NoopToolClient will be used internally.
func NewAgent(m model.Model, maxHistory int, toolsEnabled bool, toolClient tools.ToolClient) *Agent {
	if maxHistory <= 0 {
		maxHistory = 20
	}
	if toolClient == nil {
		toolClient = &tools.NoopToolClient{}
	}
	return &Agent{
		model:        m,
		toolClient:   toolClient,
		history:      make([]protocol.Message, 0),
		maxMessages:  maxHistory,
		toolsEnabled: toolsEnabled,
	}
}

// trimHistory keeps the conversation within maxMessages (excluding system message).
func (a *Agent) trimHistory() {
	if len(a.history) == 0 {
		return
	}

	systemIdx := -1
	for i, msg := range a.history {
		if msg.Role == "system" {
			systemIdx = i
			break
		}
	}

	nonSystemMsgs := a.history
	if systemIdx >= 0 {
		nonSystemMsgs = a.history[systemIdx+1:]
	}

	if len(nonSystemMsgs) > a.maxMessages {
		keepStart := len(nonSystemMsgs) - a.maxMessages
		trimmed := nonSystemMsgs[keepStart:]

		if systemIdx >= 0 {
			a.history = append([]protocol.Message{a.history[systemIdx]}, trimmed...)
		} else {
			a.history = trimmed
		}
	}
}

// Chat handles a user input and returns the assistant's response.
func (a *Agent) Chat(input string) (string, error) {
	if len(a.history) == 0 {
		systemMsg := protocol.Message{
			Role:    "system",
			Content: "You are a helpful assistant.",
		}
		a.history = append(a.history, systemMsg)
	}

	a.history = append(a.history, protocol.Message{
		Role:    "user",
		Content: input,
	})
	a.trimHistory()

	// Get tool definitions for API
	var apiTools []model.ToolForAPI
	if a.toolsEnabled {
		defs, err := a.toolClient.List()
		// fmt.Printf("提供了工具：%#v\n", defs)
		if err != nil {
			return "", fmt.Errorf("failed to list tools: %w", err)
		} else {
			apiTools = convertToolDefsToAPI(defs)
		}
	}

	// Call model with tools support
	maxRounds := 3
	for round := 0; round < maxRounds; round++ {
		var content string
		var toolCalls []protocol.ToolCall
		var err error

		content, toolCalls, err = a.model.ChatWithTools(a.history, apiTools)
		if err != nil {
			return "", fmt.Errorf("model call failed: %w", err)
		}

		// Build assistant message
		assistantMsg := protocol.Message{
			Role:      "assistant",
			Content:   content,
			ToolCalls: toolCalls,
		}
		a.history = append(a.history, assistantMsg)
		a.trimHistory()

		// If no tool calls, return final answer
		if len(toolCalls) == 0 {
			return content, nil
		}

		// Execute each tool call
		for _, tc := range toolCalls {
			// Parse arguments (tc.Function.Arguments is a JSON string)
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				a.history = append(a.history, protocol.Message{
					Role:       "tool",
					Name:       tc.Function.Name,
					ToolCallID: tc.ID,
					Content:    "Error: invalid arguments JSON",
				})
				continue
			}

			result, err := a.toolClient.Call(tc.Function.Name, args)
			if err != nil {
				a.history = append(a.history, protocol.Message{
					Role:       "tool",
					Name:       tc.Function.Name,
					ToolCallID: tc.ID,
					Content:    "Error: " + err.Error(),
				})
				continue
			}

			a.history = append(a.history, protocol.Message{
				Role:       "tool",
				Name:       tc.Function.Name,
				ToolCallID: tc.ID,
				Content:    result,
			})
		}
		a.trimHistory()
	}

	// After max rounds, return last content
	last := a.history[len(a.history)-1]
	if last.Role == "assistant" && len(last.ToolCalls) > 0 {
		return "Error: Maximum tool call depth exceeded.", nil
	}
	return last.Content, nil
}

func (a *Agent) ClearHistory() {
	a.history = make([]protocol.Message, 0)
}

func convertToolDefsToAPI(defs []tools.ToolDefinition) []model.ToolForAPI {
	apiTools := make([]model.ToolForAPI, len(defs))
	for i, def := range defs {
		props := make(map[string]interface{})
		for name, param := range def.Parameters.Properties {
			props[name] = map[string]interface{}{
				"type":        param.Type,
				"description": param.Description,
			}
		}

		schema := map[string]interface{}{
			"type":       "object",
			"properties": props,
		}
		if len(def.Parameters.Required) > 0 {
			schema["required"] = def.Parameters.Required
		}

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
