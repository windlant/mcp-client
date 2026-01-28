package main

import (
	"encoding/json"
	"fmt"

	"github.com/windlant/mcp-client/internal/protocol"
	"github.com/windlant/mcp-client/internal/tools"
	"github.com/windlant/mcp-client/internal/tools/manage/builtin"
	"github.com/windlant/mcp-client/internal/tools/manage/registry"
)

// Server handles MCP requests
type Server struct {
	reg *registry.Registry
}

// NewServer creates a new MCP server instance
func NewServer() *Server {
	reg := registry.NewRegistry()
	reg.Register(builtin.GetTimeToolDef)

	// Here you can register additional server-specific tools
	// reg.Register(someOtherToolDef)

	return &Server{
		reg: reg,
	}
}

// HandleRequest processes an MCP request and returns a raw response JSON
func (s *Server) HandleRequest(requestBytes []byte) ([]byte, error) {
	// First, unmarshal to determine the method
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
		// For call_tool, we need the name and arguments
		name, ok := rawReq["name"].(string)
		if !ok {
			return s.createErrorResponse("missing or invalid name field for call_tool")
		}

		// Extract arguments
		argsRaw, exists := rawReq["arguments"]
		if !exists {
			argsRaw = map[string]interface{}{}
		}

		argsMap, ok := argsRaw.(map[string]interface{})
		if !ok {
			return s.createErrorResponse("arguments must be an object")
		}

		// Convert to tools.ToolArguments
		args := tools.ToolArguments(argsMap)

		return s.handleCallTool(name, args)
	default:
		return s.createErrorResponse(fmt.Sprintf("unknown method: %s", method))
	}
}

// handleListTools returns the list of available tools
func (s *Server) handleListTools() ([]byte, error) {
	defs := s.reg.ListAll()

	// Filter out Function field since it shouldn't be serialized
	toolDefs := make([]tools.ToolDefinition, len(defs))
	for i, def := range defs {
		toolDefs[i] = tools.ToolDefinition{
			Name:        def.Name,
			Description: def.Description,
			Parameters:  def.Parameters,
			// Function field is omitted (not serialized)
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

// handleCallTool executes a specific tool with given arguments
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

// createErrorResponse creates a properly formatted error response
func (s *Server) createErrorResponse(message string) ([]byte, error) {
	errorResponse := protocol.MCPToolCallResponse{
		Error:  message,
		Result: "", // Ensure result is empty when there's an error
	}

	jsonBytes, err := json.Marshal(errorResponse)
	if err != nil {
		// This should never happen with our simple error structure
		// But if it does, return a basic error
		fallback := `{"error": "failed to create error response"}`
		return []byte(fallback), nil
	}

	return jsonBytes, nil
}
