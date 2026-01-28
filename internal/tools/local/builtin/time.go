package builtin

import (
	"time"

	"github.com/windlant/mcp-client/internal/tools"
)

// GetTimeTool returns the current local time as a formatted string.
// It ignores any arguments passed in.
func GetTimeTool(args tools.ToolArguments) (string, error) {
	return time.Now().Format("2006-01-28 15:04:05"), nil
}
