package registry

import "github.com/windlant/mcp-client/internal/tools"

// Registry 管理已注册的工具定义
type Registry struct {
	tools map[string]tools.ToolDefinition
}

// NewRegistry 创建一个新的工具注册表
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]tools.ToolDefinition),
	}
}

// Register 注册一个工具定义（以工具名称为键）
func (r *Registry) Register(def tools.ToolDefinition) {
	r.tools[def.Name] = def
}

// Get 根据名称获取工具定义，若不存在则返回 false
func (r *Registry) Get(name string) (tools.ToolDefinition, bool) {
	def, ok := r.tools[name]
	return def, ok
}

// ListAll 返回所有已注册的工具定义列表
func (r *Registry) ListAll() []tools.ToolDefinition {
	defs := make([]tools.ToolDefinition, 0, len(r.tools))
	for _, def := range r.tools {
		defs = append(defs, def)
	}
	return defs
}
