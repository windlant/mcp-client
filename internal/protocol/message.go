package protocol

// Message 表示与 DeepSeek 函数调用兼容的聊天消息
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	Name       string     `json:"name,omitempty"`         // 用于 tool 消息，标识工具名称
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // 助手发起的工具调用列表
	ToolCallID string     `json:"tool_call_id,omitempty"` // 工具响应对应的调用 ID
}

// ToolCall 表示模型发起的一次函数/工具调用请求
type ToolCall struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"` // 必须为 "function"
	Function Function `json:"function"`
}

// Function 描述要调用的函数
type Function struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON 编码的字符串，例如 "{\"location\": \"Beijing\"}"`
}
