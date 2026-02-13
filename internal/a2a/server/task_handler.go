package server

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/windlant/protocol/protocol/a2a_protocol"
)

// TaskHandler 处理 A2A task 创建请求
type TaskHandler struct {
	taskManager *TaskManager
}

// NewTaskHandler 创建 task handler
func NewTaskHandler(tm *TaskManager) *TaskHandler {
	return &TaskHandler{taskManager: tm}
}

// HandleCreateTask 处理 POST /tasks 请求
func (h *TaskHandler) HandleCreateTask(w http.ResponseWriter, r *http.Request) {
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

	var req a2a_protocol.CreateTaskRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if req.AgentID == "" || req.SkillID == "" {
		http.Error(w, "agentId and skillId are required", http.StatusBadRequest)
		return
	}

	// 异步执行 task，得到 taskID
	taskID, _ := h.taskManager.CreateTask(req.AgentID, req.SkillID, req.Input)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(a2a_protocol.CreateTaskResponse{TaskID: taskID})
}
