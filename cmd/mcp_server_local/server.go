package main

import (
	"encoding/json"
	"fmt"

	"github.com/windlant/mcp-client/internal/protocol"
	"github.com/windlant/mcp-client/internal/tools"
	"github.com/windlant/mcp-client/internal/tools/manage/builtin"
	"github.com/windlant/mcp-client/internal/tools/manage/registry"
)

// Server 用于处理 MCP 请求
type Server struct {
	reg *registry.Registry
}

// NewServer 创建一个新的 MCP 服务器实例
func NewServer() *Server {
	reg := registry.NewRegistry()
	reg.Register(builtin.GetTimeToolDef)

	// 在这里可以注册其他专属于服务器的工具
	// reg.Register(someOtherToolDef)

	return &Server{
		reg: reg,
	}
}

// HandleRequest 处理一个 MCP 请求，并返回原始的 JSON 响应字节
func (s *Server) HandleRequest(requestBytes []byte) ([]byte, error) {
	// 先解析 JSON，确定请求的方法类型
	var rawReq map[string]interface{}
	if err := json.Unmarshal(requestBytes, &rawReq); err != nil {
		return s.createErrorResponse(fmt.Sprintf("invalid JSON: %v", err))
	}

	method, ok := rawReq["method"].(string)
	if !ok {
		return s.createErrorResponse("missing or invalid method field")
	}

	switch method {
	case protocol.MCPMethodListTools:
		return s.handleListTools()
	case protocol.MCPMethodCallTool:
		// 对于 call_tool 请求，需要工具名称和参数
		name, ok := rawReq["name"].(string)
		if !ok {
			return s.createErrorResponse("missing or invalid name field for call_tool")
		}

		// 提取参数字段
		argsRaw, exists := rawReq["arguments"]
		if !exists {
			// 如果没有提供 arguments，默认使用空对象
			argsRaw = map[string]interface{}{}
		}

		argsMap, ok := argsRaw.(map[string]interface{})
		if !ok {
			return s.createErrorResponse("arguments must be an object")
		}

		// 转换为工具所需的参数类型
		args := tools.ToolArguments(argsMap)

		return s.handleCallTool(name, args)
	default:
		return s.createErrorResponse(fmt.Sprintf("unknown method: %s", method))
	}
}

// handleListTools 返回当前服务器支持的所有工具列表
func (s *Server) handleListTools() ([]byte, error) {
	defs := s.reg.ListAll()

	// 构造工具定义列表，注意：Function 字段不能被序列化（会变成 null）
	toolDefs := make([]tools.ToolDefinition, len(defs))
	for i, def := range defs {
		toolDefs[i] = tools.ToolDefinition{
			Name:        def.Name,
			Description: def.Description,
			Parameters:  def.Parameters,
			// Function 字段留空，因为 JSON 序列化时会忽略它（标记为 `json:"-"`）
		}
	}

	response := protocol.MCPListToolsResponse{
		Tools: toolDefs,
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return s.createErrorResponse(fmt.Sprintf("failed to marshal list_tools response: %v", err))
	}

	return jsonBytes, nil
}

// handleCallTool 执行指定名称的工具，并传入给定的参数
func (s *Server) handleCallTool(name string, args tools.ToolArguments) ([]byte, error) {
	if name == "" {
		return s.createErrorResponse("tool name is required")
	}

	def, ok := s.reg.Get(name)
	if !ok {
		return s.createErrorResponse(fmt.Sprintf("tool not found: %s", name))
	}

	result, err := def.Function(args)
	if err != nil {
		return s.createErrorResponse(fmt.Sprintf("tool execution failed: %v", err))
	}

	response := protocol.MCPToolCallResponse{
		Result: result,
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return s.createErrorResponse(fmt.Sprintf("failed to marshal call_tool response: %v", err))
	}

	return jsonBytes, nil
}

// createErrorResponse 生成一个符合协议格式的错误响应
func (s *Server) createErrorResponse(message string) ([]byte, error) {
	errorResponse := protocol.MCPToolCallResponse{
		Error:  message,
		Result: "", // 出错时确保 result 字段为空
	}

	jsonBytes, err := json.Marshal(errorResponse)
	if err != nil {
		// 理论上这个简单的结构不会序列化失败
		// 但万一失败了，就返回一个最基本的错误 JSON
		fallback := `{"error": "failed to create error response"}`
		return []byte(fallback), nil
	}

	return jsonBytes, nil
}
