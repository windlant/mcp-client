package protocol

import "github.com/windlant/mcp-client/internal/tools"

// MCP 方法常量
const (
	MCPMethodListTools = "list_tools"
	MCPMethodCallTool  = "call_tool"
)

// MCP 请求结构

type MCPListToolsRequest struct {
	Method string `json:"method"` // 必须为 "list_tools"
}

type MCPToolCallRequest struct {
	Method string                 `json:"method"` // 必须为 "call_tool"
	Name   string                 `json:"name"`
	Args   map[string]interface{} `json:"arguments"`
}

// MCP 响应结构

type MCPListToolsResponse struct {
	Tools []tools.ToolDefinition `json:"tools"`
}

type MCPToolCallResponse struct {
	Result string `json:"result"`
	Error  string `json:"error,omitempty"`
}
