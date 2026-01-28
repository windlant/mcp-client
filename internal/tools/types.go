package tools

// ToolArguments represents the input parameters for a tool call.
// It is a JSON-serializable map of key-value pairs.
type ToolArguments map[string]interface{}

// ToolFunc is the function signature that all tool implementations must follow.
// It takes arguments and returns a string result or an error.
// The result should be plain text (not JSON) for simplicity.
type ToolFunc func(ToolArguments) (string, error)
