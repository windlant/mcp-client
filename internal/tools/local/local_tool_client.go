package local

import (
	"fmt"

	"github.com/windlant/mcp-client/internal/tools"
	"github.com/windlant/mcp-client/internal/tools/manage/builtin"
	"github.com/windlant/mcp-client/internal/tools/manage/registry"
)

// LocalToolClient 是一个本地工具客户端，直接在进程内执行注册的工具
type LocalToolClient struct {
	registry *registry.Registry
}

// NewLocalToolClient 创建并初始化一个本地工具客户端，预注册内置工具（如 get_time）
func NewLocalToolClient() *LocalToolClient {
	r := registry.NewRegistry()
	r.Register(builtin.GetTimeToolDef)
	return &LocalToolClient{registry: r}
}

// Call 根据名称调用已注册的工具，并传入参数
func (c *LocalToolClient) Call(name string, args tools.ToolArguments) (string, error) {
	def, ok := c.registry.Get(name)
	if !ok {
		return "", fmt.Errorf("tool not found: %s", name)
	}
	return def.Function(args)
}

// List 返回所有已注册工具的定义列表
func (c *LocalToolClient) List() ([]tools.ToolDefinition, error) {
	return c.registry.ListAll(), nil
}

// Close 用于资源清理（本地实现无需操作）
func (c *LocalToolClient) Close() error {
	return nil
}

// GetDefinition 根据名称获取工具定义，若不存在则返回 false
func (c *LocalToolClient) GetDefinition(name string) (tools.ToolDefinition, bool) {
	return c.registry.Get(name)
}
