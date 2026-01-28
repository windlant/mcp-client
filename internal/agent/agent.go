package agent

import (
	"encoding/json"
	"fmt"

	"github.com/windlant/mcp-client/internal/model"
	"github.com/windlant/mcp-client/internal/protocol"
	"github.com/windlant/mcp-client/internal/tools"
	"github.com/windlant/mcp-client/internal/tools/local"
)

type Agent struct {
	model        model.Model
	tools        *local.LocalToolClient
	history      []protocol.Message
	maxMessages  int
	toolsEnabled bool
}

func NewAgent(m model.Model, maxHistory int, toolsEnabled bool) *Agent {
	if maxHistory <= 0 {
		maxHistory = 20
	}
	return &Agent{
		model:        m,
		tools:        local.NewLocalToolClient(),
		history:      make([]protocol.Message, 0),
		maxMessages:  maxHistory,
		toolsEnabled: toolsEnabled,
	}
}

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
		defs := a.tools.ListTools()
		apiTools = convertToolDefsToAPI(defs)
	}

	// fmt.Printf("工具列表: apiTools:%#v\n", apiTools)

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

		// fmt.Printf("工具调用: tool_call:%#v\n", toolCalls)
		// fmt.Printf("工具调用: content:%#v\n", content)
		// Execute each tool call
		for _, tc := range toolCalls {
			def, ok := a.tools.GetDefinition(tc.Function.Name)
			if !ok {
				// Report error as tool message
				a.history = append(a.history, protocol.Message{
					Role:       "tool",
					Name:       tc.Function.Name,
					ToolCallID: tc.ID,
					Content:    "Error: tool not found",
				})
				continue
			}

			// Parse arguments (tc.Function.Arguments is a JSON string)
			var args tools.ToolArguments
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				a.history = append(a.history, protocol.Message{
					Role:       "tool",
					Name:       tc.Function.Name,
					ToolCallID: tc.ID,
					Content:    "Error: invalid arguments JSON",
				})
				continue
			}

			result, err := def.Function(args)
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
		// Convert ToolSchema to JSON Schema object
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
