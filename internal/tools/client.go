package tools

import (
	"errors"
	"fmt"
	"os"

	"github.com/windlant/mcp-client/internal/config"
	"github.com/windlant/mcp-client/internal/tools/local"
	"github.com/windlant/mcp-client/internal/tools/remote"
	"github.com/windlant/mcp-client/internal/tools/stdio"
	"github.com/windlant/protocol/types/tools_types"
)

// ToolClient 是工具调用的统一接口，支持本地或远程（如 stdio、HTTP）实现
type ToolClient interface {
	// Call 调用指定名称的工具，并传入参数
	Call(name string, args tools_types.ToolArguments) (string, error)

	// List 返回所有可用工具的定义
	List() ([]tools_types.ToolDefinition, error)

	// Close 释放资源（如关闭子进程或网络连接）
	Close() error
}

// ErrToolNotFound 表示请求的工具未注册或不存在
var ErrToolNotFound = errors.New("tool not found")

// ToolSource 标识工具来源
type ToolSource string

const (
	ToolSourceRemote ToolSource = "remote"
	ToolSourceLocal  ToolSource = "local"
	ToolSourceStdio  ToolSource = "stdio"
)

// AggregatedToolClient 是一个聚合工具客户端，自动管理 remote、local 和 stdio 三种工具源
// 工具调用优先级：remote > local > stdio
type AggregatedToolClient struct {
	cfg *config.Config

	remoteClient *remoteToolWrapper
	localClient  *localToolWrapper
	stdioClient  *stdioToolWrapper

	// 总的工具列表，按优先级去重
	allTools []tools_types.ToolDefinition

	// 工具到源的映射，用于快速查找
	toolToSource map[string]ToolSource
}

// 工具包装器，包含客户端和其工具列表
type remoteToolWrapper struct {
	client ToolClient
	tools  map[string]tools_types.ToolDefinition
}

type localToolWrapper struct {
	client ToolClient
	tools  map[string]tools_types.ToolDefinition
}

type stdioToolWrapper struct {
	client ToolClient
	tools  map[string]tools_types.ToolDefinition
}

// NewAggregatedToolClient 根据配置自动创建并初始化聚合工具客户端
func NewAggregatedToolClient(cfg *config.Config) (*AggregatedToolClient, error) {
	agg := &AggregatedToolClient{
		cfg:          cfg,
		toolToSource: make(map[string]ToolSource),
	}

	// 自动初始化三种客户端
	if err := agg.initRemoteClient(); err != nil {
		return nil, fmt.Errorf("failed to initialize remote client: %w", err)
	}

	if err := agg.initLocalClient(); err != nil {
		return nil, fmt.Errorf("failed to initialize local client: %w", err)
	}

	if err := agg.initStdioClient(); err != nil {
		return nil, fmt.Errorf("failed to initialize stdio client: %w", err)
	}

	// 构建总的工具列表（按优先级去重）
	agg.buildAllTools()

	return agg, nil
}

// initRemoteClient 初始化远程客户端
func (a *AggregatedToolClient) initRemoteClient() error {
	// 检查是否配置了远程服务器
	if a.cfg.Tools.MCPServer.Host != "" && a.cfg.Tools.MCPServer.Port != 0 {
		url := fmt.Sprintf("%s:%d/mcp", a.cfg.Tools.MCPServer.Host, a.cfg.Tools.MCPServer.Port)
		client := remote.NewRemoteToolClient(url)

		tools, err := client.List()
		if err != nil {
			return fmt.Errorf("failed to list remote tools from %s: %w", url, err)
		}

		a.remoteClient = &remoteToolWrapper{
			client: client,
			tools:  toolsToMap(tools),
		}
		fmt.Printf("✓ 远程工具客户端已配置: %s\n", url)
	} else {
		fmt.Println("⚠ 远程工具客户端未配置（缺少 host 或 port）")
	}

	return nil
}

// initLocalClient 初始化本地客户端
func (a *AggregatedToolClient) initLocalClient() error {
	client := local.NewLocalToolClient()
	tools, err := client.List()
	if err != nil {
		return fmt.Errorf("failed to list local tools: %w", err)
	}

	a.localClient = &localToolWrapper{
		client: client,
		tools:  toolsToMap(tools),
	}
	fmt.Println("✓ 本地工具客户端已启用")

	return nil
}

// initStdioClient 初始化 stdio 客户端
func (a *AggregatedToolClient) initStdioClient() error {
	serverBinary := "./cmd/mcp_server_local/mcp-server-local"
	if _, err := os.Stat(serverBinary); err == nil {
		client, err := stdio.NewStdioToolClient(serverBinary)
		if err != nil {
			fmt.Printf("⚠ stdio 工具客户端启动失败: %v\n", err)
			return nil // 不返回错误，只是不启用 stdio
		}

		tools, err := client.List()
		if err != nil {
			fmt.Printf("⚠ stdio 工具客户端获取工具列表失败: %v\n", err)
			_ = client.Close() // 清理资源
			return nil         // 不返回错误，只是不启用 stdio
		}

		a.stdioClient = &stdioToolWrapper{
			client: client,
			tools:  toolsToMap(tools),
		}
		fmt.Println("✓ stdio 工具客户端已启用")
	} else {
		fmt.Printf("⚠ stdio 工具客户端未启用（二进制文件不存在: %s）\n", serverBinary)
	}

	return nil
}

// toolsToMap 将工具切片转换为 map，便于快速查找
func toolsToMap(tools []tools_types.ToolDefinition) map[string]tools_types.ToolDefinition {
	toolMap := make(map[string]tools_types.ToolDefinition)
	for _, tool := range tools {
		toolMap[tool.Name] = tool
	}
	return toolMap
}

// buildAllTools 构建总的工具列表，按优先级去重（remote > local > stdio）
func (a *AggregatedToolClient) buildAllTools() {
	toolSet := make(map[string]tools_types.ToolDefinition)

	// 按优先级顺序添加工具
	if a.remoteClient != nil {
		for name, tool := range a.remoteClient.tools {
			toolSet[name] = tool
			a.toolToSource[name] = ToolSourceRemote
		}
	}

	if a.localClient != nil {
		for name, tool := range a.localClient.tools {
			if _, exists := toolSet[name]; !exists {
				toolSet[name] = tool
				a.toolToSource[name] = ToolSourceLocal
			}
		}
	}

	if a.stdioClient != nil {
		for name, tool := range a.stdioClient.tools {
			if _, exists := toolSet[name]; !exists {
				toolSet[name] = tool
				a.toolToSource[name] = ToolSourceStdio
			}
		}
	}

	// 转换为切片
	a.allTools = make([]tools_types.ToolDefinition, 0, len(toolSet))
	for _, tool := range toolSet {
		a.allTools = append(a.allTools, tool)
	}
}

// Call 调用指定名称的工具，按照 remote > local > stdio 的优先级查找
func (a *AggregatedToolClient) Call(name string, args tools_types.ToolArguments) (string, error) {
	source, exists := a.toolToSource[name]
	if !exists {
		return "", ErrToolNotFound
	}

	switch source {
	case ToolSourceRemote:
		return a.remoteClient.client.Call(name, args)
	case ToolSourceLocal:
		return a.localClient.client.Call(name, args)
	case ToolSourceStdio:
		return a.stdioClient.client.Call(name, args)
	default:
		return "", fmt.Errorf("unknown tool source: %s", source)
	}
}

// List 返回所有可用工具的定义（去重后的总列表）
func (a *AggregatedToolClient) List() ([]tools_types.ToolDefinition, error) {
	return a.allTools, nil
}

// Close 关闭所有底层客户端
func (a *AggregatedToolClient) Close() error {
	var lastErr error

	if a.remoteClient != nil {
		if err := a.remoteClient.client.Close(); err != nil {
			lastErr = err
		}
	}

	if a.localClient != nil {
		if err := a.localClient.client.Close(); err != nil {
			lastErr = err
		}
	}

	if a.stdioClient != nil {
		if err := a.stdioClient.client.Close(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}
