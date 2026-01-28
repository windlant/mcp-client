package local

import (
	"fmt"

	"github.com/windlant/mcp-client/internal/tools"
	"github.com/windlant/mcp-client/internal/tools/local/builtin"
	"github.com/windlant/mcp-client/internal/tools/local/registry"
)

// LocalToolClient executes tools locally using a registry.
type LocalToolClient struct {
	registry *registry.Registry
}

// NewLocalToolClient creates a new local tool client with built-in tools registered.
func NewLocalToolClient() *LocalToolClient {
	r := registry.NewRegistry()
	// Register built-in tools here
	r.Register("get_current_time", builtin.GetTimeTool)
	// Example for future: r.Register("search_web", builtin.SearchWebTool)
	return &LocalToolClient{registry: r}
}

// Call executes a tool by name with the given arguments.
// Returns the tool result or an error if the tool is not found or fails.
func (c *LocalToolClient) Call(name string, args tools.ToolArguments) (string, error) {
	fn, ok := c.registry.Get(name)
	if !ok {
		return "", fmt.Errorf("tool not found: %s", name)
	}
	return fn(args)
}
