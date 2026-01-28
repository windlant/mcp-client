package registry

import "github.com/windlant/mcp-client/internal/tools"

type Registry struct {
	tools map[string]tools.ToolDefinition
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]tools.ToolDefinition),
	}
}

func (r *Registry) Register(def tools.ToolDefinition) {
	r.tools[def.Name] = def
}

func (r *Registry) Get(name string) (tools.ToolDefinition, bool) {
	def, ok := r.tools[name]
	return def, ok
}

func (r *Registry) ListAll() []tools.ToolDefinition {
	defs := make([]tools.ToolDefinition, 0, len(r.tools))
	for _, def := range r.tools {
		defs = append(defs, def)
	}
	return defs
}
