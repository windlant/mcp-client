package main

import (
	"fmt"

	"github.com/windlant/mcp-client/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	fmt.Printf("✅ 配置加载成功!\n")
	fmt.Printf("Model: %s\n", cfg.Model.ModelName)
	fmt.Printf("API Key 长度: %d\n", len(cfg.APIKey))
}
