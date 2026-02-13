package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/windlant/protocol/protocol/registry_protocol"
	"github.com/windlant/protocol/types/agent_types"
)

// RegistryClient 简单的注册中心 HTTP 客户端（只实现列表查询）
type RegistryClient struct {
	agentId    string
	baseURL    string
	httpClient *http.Client
}

// NewRegistryClient 创建 RegistryClient，baseURL 例如 "http://registry.local:8080"
func NewRegistryClient(baseURL string, agentId string) *RegistryClient {
	base := strings.TrimRight(baseURL, "/")
	return &RegistryClient{
		agentId:    agentId,
		baseURL:    base,
		httpClient: &http.Client{},
	}
}

// ListAgents 从注册中心获取所有 AgentCard
func (c *RegistryClient) ListAgents(ctx context.Context) ([]agent_types.AgentCard, error) {
	url := c.baseURL + registry_protocol.AgentsListPath
	var reqBody registry_protocol.AgentsListRequest
	reqBody.AgentID = c.agentId
	b, err := json.Marshal(reqBody)
	if err != nil {
		return []agent_types.AgentCard{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, strings.NewReader(string(b)))
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var listResp registry_protocol.AgentsListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, err
	}

	return listResp.Agents, nil
}

// Register 将本地 AgentCard 注册到注册中心，webhookURL 是用于接收通知的回调地址
func (c *RegistryClient) Register(ctx context.Context, card agent_types.AgentCard, webhookURL string) (registry_protocol.RegisterResponse, error) {
	url := c.baseURL + registry_protocol.RegisterPath

	reqBody := registry_protocol.RegisterRequest{
		AgentCard:  card,
		WebhookURL: webhookURL,
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return registry_protocol.RegisterResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(b)))
	if err != nil {
		return registry_protocol.RegisterResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return registry_protocol.RegisterResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return registry_protocol.RegisterResponse{}, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var registerResp registry_protocol.RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&registerResp); err != nil {
		return registry_protocol.RegisterResponse{}, err
	}

	return registerResp, nil
}

// Deregister 注销指定 agentID
func (c *RegistryClient) Deregister(ctx context.Context, agentID string) error {
	url := c.baseURL + registry_protocol.DeregisterPath

	reqBody := registry_protocol.DeregisterRequest{
		AgentID: agentID,
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(b)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}
