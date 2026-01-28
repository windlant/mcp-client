package local

import (
	"fmt"

	"github.com/windlant/mcp-client/internal/tools"
	"github.com/windlant/mcp-client/internal/tools/local/builtin"
	"github.com/windlant/mcp-client/internal/tools/local/registry"
)

type LocalToolClient struct {
	registry *registry.Registry
}

func NewLocalToolClient() *LocalToolClient {
	r := registry.NewRegistry()
	r.Register(builtin.GetTimeToolDef)
	return &LocalToolClient{registry: r}
}

func (c *LocalToolClient) Call(name string, args tools.ToolArguments) (string, error) {
	def, ok := c.registry.Get(name)
	if !ok {
		return "", fmt.Errorf("tool not found: %s", name)
	}
	return def.Function(args)
}

func (c *LocalToolClient) ListTools() []tools.ToolDefinition {
	return c.registry.ListAll()
}

func (c *LocalToolClient) GetDefinition(name string) (tools.ToolDefinition, bool) {
	return c.registry.Get(name)
}
