// internal/tools/client.go
package tools

import (
	"errors"
)

type ToolClient interface {
	Call(name string, args ToolArguments) (string, error)

	List() ([]ToolDefinition, error)

	Close() error
}

var ErrToolNotFound = errors.New("tool not found")
