package tools

type NoopToolClient struct{}

func (n *NoopToolClient) Call(name string, args ToolArguments) (string, error) {
	return "", ErrToolNotFound
}

func (n *NoopToolClient) List() ([]ToolDefinition, error) {
	return nil, nil
}

func (n *NoopToolClient) Close() error {
	return nil
}
