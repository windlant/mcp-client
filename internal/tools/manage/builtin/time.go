package builtin

import (
	"time"

	"github.com/windlant/protocol/types/tools_types"
)

func GetTimeTool(args tools_types.ToolArguments) (string, error) {
	return time.Now().Format("2006-01-02 15:04:05"), nil
}

var GetTimeToolDef = tools_types.ToolDefinition{
	Name:        "get_current_time",
	Description: "Get the current date and time in 'YYYY-MM-DD HH:MM:SS' format.",
	Parameters: tools_types.ToolSchema{
		Type:       "object",
		Properties: map[string]tools_types.ToolParameter{},
		Required:   []string{},
	},
	Function: GetTimeTool,
}
