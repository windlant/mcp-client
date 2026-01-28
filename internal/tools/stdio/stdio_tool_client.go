package stdio

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/windlant/mcp-client/internal/protocol"
	"github.com/windlant/mcp-client/internal/tools"
)

// StdioToolClient 通过子进程的 stdin/stdout 与 MCP 工具服务器通信
type StdioToolClient struct {
	cmd    *exec.Cmd
	stdin  *json.Encoder
	stdout *bufio.Scanner
	mu     sync.Mutex // 确保请求-响应交互是线程安全的，之后可能有多个client通过stdio访问server
}

// NewStdioToolClient 启动一个 MCP 服务器子进程，并建立通信管道
func NewStdioToolClient(serverBinary string) (*StdioToolClient, error) {
	cmd := exec.Command(serverBinary)

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start server process: %w", err)
	}

	client := &StdioToolClient{
		cmd:    cmd,
		stdin:  json.NewEncoder(stdinPipe),
		stdout: bufio.NewScanner(stdoutPipe),
	}

	return client, nil
}

// sendRequest 向子进程发送请求并等待单行 JSON 响应（NDJSON 格式）
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

// Call 调用指定名称的工具，并传入参数
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

// List 获取服务器支持的所有工具定义
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

	return resp.Tools, nil
}

// Close 优雅关闭子进程：先发送中断信号，超时后强制终止
func (c *StdioToolClient) Close() error {
	if c.cmd.Process == nil {
		return nil
	}

	_ = c.cmd.Process.Signal(os.Interrupt)

	done := make(chan error, 1)
	go func() {
		done <- c.cmd.Wait()
	}()

	select {
	case <-done:
		return nil
	case <-time.After(2 * time.Second):
		_ = c.cmd.Process.Kill()
		_ = c.cmd.Wait()
		return nil
	}
}
