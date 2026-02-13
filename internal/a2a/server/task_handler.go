package server

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

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

// HandleCreateTask 处理 POST /tasks 请求，创建异步任务并立即返回 taskID
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

	// 设置响应头并返回 CreateTaskResponse（包含 taskID）
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(a2a_protocol.CreateTaskResponse{TaskID: taskID})
}

// HandleGetTaskResult 处理 GET /tasks/{taskID} 请求，阻塞等待任务结果
func (h *TaskHandler) HandleGetTaskResult(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 从 URL 中提取 taskID (mux 应该已经解析)
	taskID := r.PathValue("taskID")
	if taskID == "" {
		http.Error(w, "taskID is required", http.StatusBadRequest)
		return
	}

	// 阻塞等待任务完成（使用 10 分钟超时）
	result, err := h.taskManager.GetTaskResult(taskID, 10*time.Minute)

	// 设置响应头
	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		// 任务获取失败，返回错误响应
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(a2a_protocol.TaskResult{
			TaskID:   taskID,
			Success:  false,
			Artifact: nil,
			Error:    err.Error(),
		})
		h.taskManager.CleanupCompletedTask(taskID)
		return
	}

	// 返回完整的 TaskResult
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result)

	// 清理已完成的任务
	h.taskManager.CleanupCompletedTask(taskID)
}
