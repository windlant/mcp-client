package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/windlant/protocol/protocol/a2a_protocol"
	"github.com/windlant/protocol/types/agent_types"
	"github.com/windlant/protocol/types/skill_types"
)

// A2AClient 用于向远程 agent 发送 task 请求
type A2AClient struct {
	httpClient *http.Client
}

// NewA2AClient 创建新的 A2A client，设置 10 分钟超时
func NewA2AClient() *A2AClient {
	return &A2AClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}
}

// CreateTask 向目标 agent 发送 task 请求，先接收 taskID，然后阻塞等待任务结果
func (c *A2AClient) CreateTask(ctx context.Context, targetAgent agent_types.AgentCard, skillID string, input skill_types.SkillArguments) (*a2a_protocol.TaskResult, error) {
	// 构造 CreateTaskRequest
	req := a2a_protocol.CreateTaskRequest{
		AgentID: targetAgent.AgentID,
		SkillID: skillID,
		Input:   input,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 发送 POST 请求到目标 agent 的 /tasks 端点
	url := targetAgent.URL + a2a_protocol.TasksPath
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// 先接收 CreateTaskResponse（包含 taskID）
	var createResp a2a_protocol.CreateTaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		return nil, fmt.Errorf("failed to decode task response: %w", err)
	}
	log.Printf("Task created with ID: %s\n", createResp.TaskID)

	// 再接收 TaskResult（服务器阻塞等待任务完成后返回）
	var result a2a_protocol.TaskResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode task result: %w", err)
	}

	log.Printf("Task %s completed: %v\n", result.TaskID, result)
	return &result, nil
}

// SendMessage 向目标 agent 发送消息（后期用于 ask 工具等）
func (c *A2AClient) SendMessage(ctx context.Context, targetAgent agent_types.AgentCard, message string) (*a2a_protocol.SendMessageResponse, error) {
	req := a2a_protocol.SendMessageRequest{
		ToAgentID: targetAgent.AgentID,
		Message:   message,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	url := targetAgent.URL + a2a_protocol.MessagesPath
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result a2a_protocol.SendMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}
