package builtin

import (
	"time"

	"github.com/windlant/mcp-client/internal/tools"
)

func GetTimeTool(args tools.ToolArguments) (string, error) {
	return time.Now().Format("2006-01-02 15:04:05"), nil
}

var GetTimeToolDef = tools.ToolDefinition{
	Name:        "get_current_time",
	Description: "Get the current date and time in 'YYYY-MM-DD HH:MM:SS' format.",
	Parameters: tools.ToolSchema{
		Type:       "object",
		Properties: map[string]tools.ToolParameter{},
		Required:   []string{},
	},
	Function: GetTimeTool,
}
