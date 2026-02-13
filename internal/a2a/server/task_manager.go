package server

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/windlant/mcp-client/internal/agent"
	"github.com/windlant/mcp-client/internal/skills"
	"github.com/windlant/protocol/protocol/a2a_protocol"
	"github.com/windlant/protocol/types/skill_types"
)

// TaskManager 管理异步 task 执行，支持并发限制
type TaskManager struct {
	mu             sync.Mutex
	tasks          map[string]*TaskExecution // TaskID -> 执行状态
	skillClient    *skills.IntegratedClient
	maxConcurrency int
	semaphore      chan struct{}
	taskIDCounter  int64
	agentFactory   func() *agent.Agent
}

// TaskExecution 表示单个 task 的执行状态
type TaskExecution struct {
	TaskID      string
	AgentID     string
	SkillID     string
	Input       skill_types.SkillArguments
	Result      *a2a_protocol.TaskResult
	Done        chan struct{}
	StartedAt   time.Time
	CompletedAt time.Time
	Agent       *agent.Agent
}

// NewTaskManager 创建 task manager，maxConcurrency 设定最大并行 task 数
func NewTaskManager(skillClient *skills.IntegratedClient, maxConcurrency int, agentFactory func() *agent.Agent) *TaskManager {
	if maxConcurrency <= 0 {
		maxConcurrency = 10
	}
	return &TaskManager{
		tasks:          make(map[string]*TaskExecution),
		skillClient:    skillClient,
		maxConcurrency: maxConcurrency,
		semaphore:      make(chan struct{}, maxConcurrency),
		agentFactory:   agentFactory,
	}
}

// CreateTask 创建并异步执行一个 task，返回 TaskID 与初始响应
func (tm *TaskManager) CreateTask(agentID, skillID string, input skill_types.SkillArguments) (string, *a2a_protocol.TaskResult) {
	tm.mu.Lock()
	tm.taskIDCounter++
	taskID := fmt.Sprintf("task-%d", tm.taskIDCounter)
	tm.mu.Unlock()

	exec := &TaskExecution{
		TaskID:    taskID,
		AgentID:   agentID,
		SkillID:   skillID,
		Input:     input,
		Done:      make(chan struct{}),
		StartedAt: time.Now(),
	}

	// 如果提供了 agentFactory，为该 task 创建独立的 Agent 并绑定
	if tm.agentFactory != nil {
		exec.Agent = tm.agentFactory()
	}

	tm.mu.Lock()
	tm.tasks[taskID] = exec
	tm.mu.Unlock()

	// 异步执行 task
	go tm.executeTask(exec)

	// 立即返回 TaskID（同步响应）
	return taskID, &a2a_protocol.TaskResult{
		TaskID:   taskID,
		Success:  true,
		Artifact: nil,
		Error:    "",
	}
}

// executeTask 实际执行 task，运行在独立的 goroutine 中
func (tm *TaskManager) executeTask(exec *TaskExecution) {
	// 获取信号量（控制并发）
	tm.semaphore <- struct{}{}
	defer func() { <-tm.semaphore }()

	// 调用本地 skill client 执行 skill
	// 将本 task 的 Agent 放入 context，以便 executor 使用该 Agent（或由 executor 创建短生命周期 Agent）
	ctx := context.Background()
	if exec.Agent != nil {
		ctx = context.WithValue(ctx, skills.ContextAgentKey, exec.Agent)
	}
	result, err := tm.skillClient.Call(ctx, exec.SkillID, exec.Input)
	exec.CompletedAt = time.Now()

	var artifact interface{}
	var errMsg string

	if err != nil {
		artifact = nil
		errMsg = err.Error()
		log.Printf("task %s skill execution error: %v", exec.TaskID, err)
	} else {
		artifact = result
		errMsg = ""
	}

	// 存储结果
	exec.Result = &a2a_protocol.TaskResult{
		TaskID:   exec.TaskID,
		Success:  err == nil,
		Artifact: artifact,
		Error:    errMsg,
	}

	close(exec.Done)
	log.Printf("task %s completed: success=%v", exec.TaskID, exec.Result.Success)
}

// GetTaskResult 获取指定 taskID 的结果，会阻塞直到任务完成或超时
func (tm *TaskManager) GetTaskResult(taskID string, timeout time.Duration) (*a2a_protocol.TaskResult, error) {
	tm.mu.Lock()
	exec, ok := tm.tasks[taskID]
	tm.mu.Unlock()

	if !ok {
		return nil, fmt.Errorf("task not found")
	}

	// 等待任务完成或超时
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	select {
	case <-exec.Done:
		return exec.Result, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("task timeout after %v", timeout)
	}
}

// CleanupCompletedTask 清理已完成的 task（可选，防止内存泄漏）
func (tm *TaskManager) CleanupCompletedTask(taskID string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	delete(tm.tasks, taskID)
}
