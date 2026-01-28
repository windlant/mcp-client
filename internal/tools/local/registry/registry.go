package registry

import "github.com/windlant/mcp-client/internal/tools"

// Registry stores and manages available tools by name.
type Registry struct {
	tools map[string]tools.ToolFunc
}

// NewRegistry creates a new empty tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]tools.ToolFunc),
	}
}

// Register adds a tool function to the registry under the given name.
func (r *Registry) Register(name string, fn tools.ToolFunc) {
	r.tools[name] = fn
}

// Get retrieves a tool function by name.
// Returns the function and true if found; otherwise returns nil and false.
func (r *Registry) Get(name string) (tools.ToolFunc, bool) {
	fn, ok := r.tools[name]
	return fn, ok
}
