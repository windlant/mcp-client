package protocol

import "github.com/windlant/mcp-client/internal/tools"

// MCP Method Constants
const (
	MCPMethodListTools = "list_tools"
	MCPMethodCallTool  = "call_tool"
)

// MCP Requests

type MCPListToolsRequest struct {
	Method string `json:"method"` // must be "list_tools"
}

type MCPToolCallRequest struct {
	Method string                 `json:"method"` // must be "call_tool"
	Name   string                 `json:"name"`
	Args   map[string]interface{} `json:"arguments"`
}

// MCP Responses

type MCPListToolsResponse struct {
	Tools []tools.ToolDefinition `json:"tools"`
}

type MCPToolCallResponse struct {
	Result string `json:"result"`
	Error  string `json:"error,omitempty"`
}
