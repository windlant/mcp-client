package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/windlant/mcp-client/internal/config"
	"github.com/windlant/mcp-client/internal/protocol"
)

// DeepSeekModel 是对接 DeepSeek API 的模型实现
type DeepSeekModel struct {
	apiKey     string
	modelName  string
	httpClient *http.Client
}

// NewDeepSeekModel 根据配置创建 DeepSeek 模型实例
func NewDeepSeekModel(cfg *config.Config) (Model, error) {
	if cfg.Model.APIKey == "" {
		return nil, fmt.Errorf("DeepSeek API key is required")
	}
	return &DeepSeekModel{
		apiKey:    cfg.Model.APIKey,
		modelName: cfg.Model.ModelName,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

// Chat 发送普通对话消息（不使用工具），返回模型的文本回复
func (d *DeepSeekModel) Chat(messages []protocol.Message) (string, error) {
	// 准备请求体
	reqBody := map[string]interface{}{
		"model":    d.modelName,
		"messages": messages,
		"stream":   false,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", "https://api.deepseek.com/chat/completions", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+d.apiKey)

	// 发送请求
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应内容
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// 检查 HTTP 状态码
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("DeepSeek API error (%d): %s", resp.StatusCode, string(respBody))
	}

	// 解析 API 响应
	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse DeepSeek response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from DeepSeek")
	}

	return apiResp.Choices[0].Message.Content, nil
}

// ChatWithTools 发送支持工具调用的对话请求，返回文本内容和工具调用列表
func (d *DeepSeekModel) ChatWithTools(messages []protocol.Message, tools []ToolForAPI) (string, []protocol.ToolCall, error) {
	reqBody := map[string]interface{}{
		"model":    d.modelName,
		"messages": messages,
		"stream":   false,
	}
	if len(tools) > 0 {
		reqBody["tools"] = tools
		reqBody["tool_choice"] = "auto"
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.deepseek.com/chat/completions", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+d.apiKey)

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("DeepSeek API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var apiResp struct {
		Choices []struct {
			Message struct {
				Content   string              `json:"content"`
				ToolCalls []protocol.ToolCall `json:"tool_calls,omitempty"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return "", nil, fmt.Errorf("no choices in response")
	}

	msg := apiResp.Choices[0].Message
	content := msg.Content
	if content == "" && len(msg.ToolCalls) > 0 {
		content = "{}" // 占位符；实际关注的是 ToolCalls
	}

	return content, msg.ToolCalls, nil
}
