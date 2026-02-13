package server

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/windlant/mcp-client/internal/a2a/registry"
	"github.com/windlant/mcp-client/internal/skills"
	"github.com/windlant/protocol/protocol/registry_protocol"
)

// WebhookHandler 接收来自注册中心的 AgentUpdateNotification
type WebhookHandler struct {
	registrar   *registry.Registrar
	skillClient *skills.IntegratedClient // 整合客户端，用于维护远程 skills 缓存
}

// NewWebhookHandler 创建新的 webhook 处理器
func NewWebhookHandler(reg *registry.Registrar, sc *skills.IntegratedClient) *WebhookHandler {
	return &WebhookHandler{registrar: reg, skillClient: sc}
}

// HandleNotification 作为 HTTP handler 接收 POST 通知
func (h *WebhookHandler) HandleNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var notif registry_protocol.AgentUpdateNotification
	if err := json.Unmarshal(body, &notif); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	switch notif.Type {
	case registry_protocol.NotificationTypeAgentRegistered, registry_protocol.NotificationTypeAgentUpdated:
		if notif.AgentCard != nil {
			h.registrar.Update(*notif.AgentCard)
			// 同时更新 skillClient 中的远程 skills
			if h.skillClient != nil {
				h.skillClient.RegisterRemoteAgentSkills(*notif.AgentCard)
			}
			// log.Printf("Webhook: updated agent %s", notif.AgentID)
		}
	case registry_protocol.NotificationTypeAgentDeregistered:
		h.registrar.Remove(notif.AgentID)
		// 同时下线远程 agent 的 skills
		if h.skillClient != nil {
			h.skillClient.UnregisterRemoteAgentSkills(notif.AgentID)
		}
		log.Printf("Webhook: removed agent %s", notif.AgentID)
	default:
		log.Printf("Webhook: unknown notification type %s", notif.Type)
	}

	w.WriteHeader(http.StatusOK)
}
