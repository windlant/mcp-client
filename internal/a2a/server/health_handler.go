package server

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/windlant/mcp-client/internal/a2a/registry"
	"github.com/windlant/protocol/protocol/registry_protocol"
)

// HealthHandler 处理注册中心的健康检查请求
type HealthHandler struct {
	registrar *registry.Registrar
}

func NewHealthHandler(reg *registry.Registrar) *HealthHandler {
	return &HealthHandler{registrar: reg}
}

// HandleHealth 响应 AgentHealthCheckRequest
func (h *HealthHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
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

	var req registry_protocol.AgentHealthCheckRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	resp := registry_protocol.AgentHealthCheckResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		AgentID:   req.AgentID,
		Message:   "",
	}

	// if req.AgentID != "" {
	// 	if card, ok := h.registrar.Get(req.AgentID); ok {
	// 		resp.AgentCard = &card
	// 	}
	// }

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("failed to encode health response: %v", err)
	}
}
