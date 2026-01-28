package agent

import (
	"encoding/json"
	"fmt"
	"strings"

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

func (a *Agent) buildSystemMessage() string {
	if !a.toolsEnabled {
		return "You are a helpful assistant."
	}

	toolDefs := a.tools.ListTools()
	if len(toolDefs) == 0 {
		return "You are a helpful assistant."
	}

	var b strings.Builder
	b.WriteString("You are a helpful assistant that can use the following tools:\n\n")

	for _, def := range toolDefs {
		b.WriteString(fmt.Sprintf("Tool: %s\n", def.Name))
		b.WriteString(fmt.Sprintf("Description: %s\n", def.Description))

		if len(def.Parameters.Properties) > 0 {
			b.WriteString("Parameters:\n")
			for name, param := range def.Parameters.Properties {
				requiredMark := ""
				if param.Required {
					requiredMark = " [required]"
				}
				b.WriteString(fmt.Sprintf("  - %s (%s): %s%s\n", name, param.Type, param.Description, requiredMark))
			}
		} else {
			b.WriteString("Parameters: none\n")
		}
		b.WriteString("\n")
	}

	b.WriteString(`When you need to use a tool, respond ONLY with a JSON object in this exact format:
{
  "tools": [
    {
      "name": "<tool_name>",
      "arguments": {<key>: <value>, ...}
    }
  ]
}
Do not include any other text or explanation.`)

	return b.String()
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
			Content: a.buildSystemMessage(),
		}
		a.history = append(a.history, systemMsg)
	}

	a.history = append(a.history, protocol.Message{
		Role:    "user",
		Content: input,
	})

	a.trimHistory()

	maxRounds := 5
	for round := 0; round < maxRounds; round++ {
		responseText, err := a.model.Chat(a.history)
		if err != nil {
			return "", fmt.Errorf("model call failed: %w", err)
		}

		assistantMsg := protocol.Message{
			Role:    "assistant",
			Content: responseText,
		}
		a.history = append(a.history, assistantMsg)
		a.trimHistory()

		if !a.isValidToolCallFormat(responseText) {
			return responseText, nil
		}

		var wrapper struct {
			Tools []struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments"`
			} `json:"tools"`
		}

		if err := json.Unmarshal([]byte(responseText), &wrapper); err != nil {
			return responseText, nil
		}

		var hasError bool
		for _, call := range wrapper.Tools {
			def, ok := a.tools.GetDefinition(call.Name)
			if !ok {
				hasError = true
				a.history = append(a.history, protocol.Message{
					Role:    "tool",
					Name:    call.Name,
					Content: fmt.Sprintf("Error: tool '%s' not found", call.Name),
				})
				continue
			}

			result, err := def.Function(tools.ToolArguments(call.Arguments))
			if err != nil {
				hasError = true
				a.history = append(a.history, protocol.Message{
					Role:    "tool",
					Name:    call.Name,
					Content: fmt.Sprintf("Error: %v", err),
				})
				continue
			}

			a.history = append(a.history, protocol.Message{
				Role:    "tool",
				Name:    call.Name,
				Content: result,
			})
		}
		a.trimHistory()

		if hasError {
			continue
		}
	}

	lastMsg := a.history[len(a.history)-1]
	if a.isValidToolCallFormat(lastMsg.Content) {
		errMsg := "Error: Maximum tool call depth exceeded. Please simplify your request."
		return errMsg, nil
	}

	return lastMsg.Content, nil
}

func (a *Agent) isValidToolCallFormat(content string) bool {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "{") || !strings.Contains(content, `"tools"`) {
		return false
	}
	var tmp struct {
		Tools interface{} `json:"tools"`
	}
	return json.Unmarshal([]byte(content), &tmp) == nil
}

func (a *Agent) ClearHistory() {
	a.history = make([]protocol.Message, 0)
}
