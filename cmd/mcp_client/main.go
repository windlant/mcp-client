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
	"github.com/windlant/mcp-client/internal/tools/local"
	"github.com/windlant/mcp-client/internal/tools/stdio"
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

	// 根据配置选择工具调用模式：local（直接调用）或 stdio（子进程服务器）
	if cfg.Tools.Enabled {
		switch cfg.Tools.Mode {
		case "local":
			tc = local.NewLocalToolClient()
			fmt.Println("使用本地工具客户端（直接函数调用）。")

		case "stdio":
			serverBinary := "./cmd/mcp_server_local/mcp-server-local"
			tc, err = stdio.NewStdioToolClient(serverBinary)
			if err != nil {
				log.Fatalf("启动 stdio 工具客户端失败: %v", err)
			}
			defer func() {
				_ = tc.Close()
			}()
			fmt.Println("使用 stdio 工具客户端（子进程 MCP 服务器）。")

		default:
			log.Fatalf("不支持的工具模式: %s。支持的模式: local, stdio", cfg.Tools.Mode)
		}
	}

	a := agent.NewAgent(m, cfg.Context.MaxHistory, cfg.Tools.Enabled, tc)

	fmt.Println("MCP 客户端已启动！")
	if cfg.Tools.Enabled {
		fmt.Printf("工具调用: 已启用 (模式: %s)\n", cfg.Tools.Mode)
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
