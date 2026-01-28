package tools

import (
	"errors"
)

// ToolClient 是工具调用的统一接口，支持本地或远程（如 stdio、HTTP）实现
type ToolClient interface {
	// Call 调用指定名称的工具，并传入参数
	Call(name string, args ToolArguments) (string, error)

	// List 返回所有可用工具的定义
	List() ([]ToolDefinition, error)

	// Close 释放资源（如关闭子进程或网络连接）
	Close() error
}

// ErrToolNotFound 表示请求的工具未注册或不存在
var ErrToolNotFound = errors.New("tool not found")
