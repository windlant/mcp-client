package stdio

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os/exec"
	"sync"

	"github.com/windlant/mcp-client/internal/protocol"
	"github.com/windlant/mcp-client/internal/tools"
)

type StdioToolClient struct {
	cmd    *exec.Cmd
	stdin  *json.Encoder
	stdout *bufio.Scanner
	mu     sync.Mutex // ensure thread-safe calls
}

// NewStdioToolClient starts the mcp-server-local subprocess and sets up communication.
func NewStdioToolClient(serverBinary string) (*StdioToolClient, error) {
	cmd := exec.Command(serverBinary)

	// Create pipes
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Start the subprocess
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start server process: %w", err)
	}

	client := &StdioToolClient{
		cmd:    cmd,
		stdin:  json.NewEncoder(stdinPipe),
		stdout: bufio.NewScanner(stdoutPipe),
	}

	// Optionally: verify server is ready? (We assume it responds immediately)
	return client, nil
}

// sendRequest sends a request and reads one line of response.
func (c *StdioToolClient) sendRequest(req interface{}) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.stdin.Encode(req); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if c.stdout.Scan() {
		return c.stdout.Bytes(), nil
	}

	if err := c.stdout.Err(); err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	return nil, fmt.Errorf("server closed stdout unexpectedly")
}

// Call invokes a tool by name with arguments.
func (c *StdioToolClient) Call(name string, args tools.ToolArguments) (string, error) {
	req := protocol.MCPToolCallRequest{
		Method: protocol.MCPMethodCallTool,
		Name:   name,
		Args:   args,
	}

	respBytes, err := c.sendRequest(req)
	if err != nil {
		return "", err
	}

	var resp protocol.MCPToolCallResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return "", fmt.Errorf("failed to parse tool call response: %w", err)
	}

	if resp.Error != "" {
		return "", fmt.Errorf("tool error: %s", resp.Error)
	}

	return resp.Result, nil
}

// List retrieves all available tools from the server.
func (c *StdioToolClient) List() ([]tools.ToolDefinition, error) {
	req := protocol.MCPListToolsRequest{
		Method: protocol.MCPMethodListTools,
	}

	respBytes, err := c.sendRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send list_tools request: %w", err)
	}

	var resp protocol.MCPListToolsResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse list_tools response: %w", err)
	}
	// fmt.Printf("返回的工具：%#v\n", resp)

	return resp.Tools, nil
}

func (c *StdioToolClient) Close() error {
	if c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
	}
	return c.cmd.Wait()
}
