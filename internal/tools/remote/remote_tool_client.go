package remote

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/windlant/protocol/protocol/mcp_protocol"
	"github.com/windlant/protocol/types/tools_types"
)

// RemoteToolClient 通过 HTTP 与远程 MCP 服务器通信
type RemoteToolClient struct {
	url     string
	agentID string // 新增：Agent ID 用于权限控制
	client  *http.Client
}

// NewRemoteToolClient 创建一个新的远程工具客户端
// url 参数应该是完整的 MCP 服务器端点 URL，例如 "http://localhost:44444/mcp"
// agentID 是当前 Agent 的唯一标识符，用于 MCP 服务器的权限控制
func NewRemoteToolClient(url string, agentID string) *RemoteToolClient {
	return &RemoteToolClient{
		url:     url,
		agentID: agentID,
		client: &http.Client{
			Timeout: 30 * time.Second, // 设置合理的超时时间
		},
	}
}

// Call 调用指定名称的远程工具，并传入参数
func (c *RemoteToolClient) Call(name string, args tools_types.ToolArguments) (string, error) {
	req := mcp_protocol.MCPToolCallRequest{
		Method:  mcp_protocol.MCPMethodCallTool,
		Name:    name,
		Args:    args,
		AgentID: c.agentID, // 添加 agent_id 到请求中
	}

	respBytes, err := c.sendRequest(req)
	if err != nil {
		return "", fmt.Errorf("failed to call remote tool %s: %w", name, err)
	}

	var resp mcp_protocol.MCPToolCallResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return "", fmt.Errorf("failed to parse tool call response: %w", err)
	}

	if resp.Error != "" {
		return "", fmt.Errorf("remote tool error: %s", resp.Error)
	}

	return resp.Result, nil
}

// List 获取远程服务器支持的所有工具定义
func (c *RemoteToolClient) List() ([]tools_types.ToolDefinition, error) {
	req := mcp_protocol.MCPListToolsRequest{
		Method:  mcp_protocol.MCPMethodListTools,
		AgentID: c.agentID, // 添加 agent_id 到请求中
	}

	respBytes, err := c.sendRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list remote tools: %w", err)
	}

	var resp mcp_protocol.MCPListToolsResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse list_tools response: %w", err)
	}

	return resp.Tools, nil
}

// Close 用于资源清理（HTTP 客户端通常无需特殊清理）
func (c *RemoteToolClient) Close() error {
	// http.Client 在 Go 中通常是可重用的，不需要显式关闭
	// 如果需要更复杂的连接管理，可以在这里实现
	return nil
}

// sendRequest 向远程服务器发送 JSON 请求并返回响应
func (c *RemoteToolClient) sendRequest(req interface{}) ([]byte, error) {
	// 序列化请求
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建 HTTP 请求
	httpReq, err := http.NewRequest("POST", c.url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// 发送请求
	httpResp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer httpResp.Body.Close()

	// 检查 HTTP 状态码
	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", httpResp.StatusCode, string(body))
	}

	// 读取响应体
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return respBody, nil
}
