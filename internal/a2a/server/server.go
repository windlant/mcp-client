package server

import (
	"log"
	"net/http"

	"github.com/windlant/mcp-client/internal/a2a/registry"
	"github.com/windlant/mcp-client/internal/agent"
	"github.com/windlant/mcp-client/internal/skills"
)

// Start 启动一个最小的 A2A HTTP 服务器并挂载注册中心 webhook 路由
// registrar 必须由调用方创建并在外部持有，以便在注册完成后更新本地缓存
// skillClient 用于本地 skill 执行
func Start(addr string, registrar *registry.Registrar, skillClient *skills.IntegratedClient, agentFactory func() *agent.Agent) *http.Server {
	wh := NewWebhookHandler(registrar, skillClient)
	hh := NewHealthHandler(registrar)
	tm := NewTaskManager(skillClient, 10, agentFactory) // 最多并行 10 个 task
	th := NewTaskHandler(tm)

	mux := http.NewServeMux()
	mux.HandleFunc("/registry/webhook", wh.HandleNotification)
	mux.HandleFunc("/health", hh.HandleHealth)
	mux.HandleFunc("POST /tasks", th.HandleCreateTask)
	mux.HandleFunc("GET /tasks/{taskID}", th.HandleGetTaskResult)

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		log.Printf("a2a server listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("a2a server error: %v", err)
		}
	}()

	return srv
}
