package tools

import "github.com/windlant/protocol/types/tools_types"

// NoopToolClient 是一个空操作的工具客户端，用于禁用工具调用的场景
type NoopToolClient struct{}

// Call 始终返回 ErrToolNotFound，表示无可用工具
func (n *NoopToolClient) Call(name string, args tools_types.ToolArguments) (string, error) {
	return "", ErrToolNotFound
}

// List 返回空的工具列表
func (n *NoopToolClient) List() ([]tools_types.ToolDefinition, error) {
	return nil, nil
}

// Close 无需清理资源
func (n *NoopToolClient) Close() error {
	return nil
}
