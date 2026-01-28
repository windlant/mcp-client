package tools

// ToolArguments represents the input parameters for a tool call.
type ToolArguments map[string]interface{}

// ToolFunc is the function signature that all tool implementations must follow.
type ToolFunc func(ToolArguments) (string, error)

// ToolParameter describes a single parameter of a tool.
type ToolParameter struct {
	Type        string `json:"type"`        // e.g., "string", "number", "object"
	Description string `json:"description"` // What this parameter is for
	Required    bool   `json:"required"`    // Whether this parameter is required
}

// ToolSchema describes the expected input structure of a tool.
type ToolSchema struct {
	Type       string                   `json:"type"`       // Usually "object"
	Properties map[string]ToolParameter `json:"properties"` // Parameter definitions
	Required   []string                 `json:"required"`   // List of required param names
}

// ToolDefinition contains all metadata needed for an LLM to use a tool.
type ToolDefinition struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Parameters  ToolSchema `json:"parameters"`
	Function    ToolFunc   `json:"-"` // Not serialized; used only locally
}
