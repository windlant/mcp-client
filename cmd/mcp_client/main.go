package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/chzyer/readline"
	"github.com/windlant/mcp-client/internal/agent"
	"github.com/windlant/mcp-client/internal/config"
	"github.com/windlant/mcp-client/internal/model"
	"github.com/windlant/mcp-client/internal/tools"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		fmt.Fprintln(os.Stderr, "请确保 'config/config.yaml' 文件存在。")
		os.Exit(1)
	}

	m, err := model.NewDeepSeekModel(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化模型失败: %v\n", err)
		os.Exit(1)
	}

	var tc tools.ToolClient

	// 如果工具启用，创建聚合工具客户端
	if cfg.Tools.Enabled {
		var err error
		tc, err = tools.NewAggregatedToolClient(cfg)
		if err != nil {
			log.Fatalf("创建聚合工具客户端失败: %v", err)
		}

		defer func() {
			_ = tc.Close()
		}()

		// 显示工具信息
		if toolsList, err := tc.List(); err == nil {
			fmt.Printf("✓ 已加载 %d 个工具（优先级: remote > local > stdio）\n", len(toolsList))
		}
	}

	a := agent.NewAgent(m, cfg.Context.MaxHistory, cfg.Tools.Enabled, tc)

	fmt.Println("MCP 客户端已启动！")
	if cfg.Tools.Enabled {
		fmt.Println("工具调用: 已启用（聚合模式）")
	} else {
		fmt.Println("工具调用: 已禁用")
	}
	fmt.Printf("最大上下文消息数: %d\n", cfg.Context.MaxHistory)
	fmt.Println("输入 'exit' 退出，输入 'clear' 清空对话历史。")

	rl, err := readline.New("You: ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化输入读取器失败: %v\n", err)
		os.Exit(1)
	}
	defer rl.Close()

	// 主交互循环：不断读取用户输入并让智能体回复
	for {
		line, err := rl.Readline()
		if err != nil {
			// 处理 EOF（Ctrl+D）或读取错误
			break
		}

		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}

		// 处理内置命令
		switch input {
		case "exit":
			fmt.Println("再见！")
			return
		case "clear":
			a.ClearHistory()
			fmt.Println("对话历史已清空。")
			continue
		}

		// 将用户输入交给智能体处理
		reply, err := a.Chat(input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "处理请求时出错: %v\n", err)
			continue
		}

		fmt.Printf("Agent: %s\n\n", reply)
	}
}
