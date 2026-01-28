// internal/tools/local/local.go
package local

import (
	"fmt"

	"github.com/windlant/mcp-client/internal/tools"
	"github.com/windlant/mcp-client/internal/tools/manage/builtin"
	"github.com/windlant/mcp-client/internal/tools/manage/registry"
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

func (c *LocalToolClient) List() ([]tools.ToolDefinition, error) {
	return c.registry.ListAll(), nil
}

func (c *LocalToolClient) Close() error {
	return nil
}

func (c *LocalToolClient) GetDefinition(name string) (tools.ToolDefinition, bool) {
	return c.registry.Get(name)
}
