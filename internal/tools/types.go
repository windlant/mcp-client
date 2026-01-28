package tools

// ToolArguments 表示工具调用的输入参数
type ToolArguments map[string]interface{}

// ToolFunc 是所有工具实现必须遵循的函数签名
type ToolFunc func(ToolArguments) (string, error)

// ToolParameter 描述工具的一个参数
type ToolParameter struct {
	Type        string `json:"type"`        // 例如 "string"、"number"、"object"
	Description string `json:"description"` // 参数用途说明
	Required    bool   `json:"required"`    // 该参数是否必需（仅用于文档，实际 required 列表在 Schema 中）
}

// ToolSchema 描述工具期望的输入结构（遵循 JSON Schema 规范）
type ToolSchema struct {
	Type       string                   `json:"type"`       // 通常为 "object"
	Properties map[string]ToolParameter `json:"properties"` // 参数定义
	Required   []string                 `json:"required"`   // 必需参数的名称列表
}

// ToolDefinition 包含 LLM 使用工具所需的全部元数据
type ToolDefinition struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Parameters  ToolSchema `json:"parameters"`
	Function    ToolFunc   `json:"-"` // 不参与 JSON 序列化，仅在本地执行时使用
}
