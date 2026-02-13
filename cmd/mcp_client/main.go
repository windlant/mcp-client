package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/chzyer/readline"
	a2aclient "github.com/windlant/mcp-client/internal/a2a/client"
	a2aregistry "github.com/windlant/mcp-client/internal/a2a/registry"
	a2aserver "github.com/windlant/mcp-client/internal/a2a/server"
	"github.com/windlant/mcp-client/internal/agent"
	"github.com/windlant/mcp-client/internal/config"
	"github.com/windlant/mcp-client/internal/model"
	"github.com/windlant/mcp-client/internal/skills"
	"github.com/windlant/mcp-client/internal/tools"
	"github.com/windlant/protocol/types/agent_types"
	"github.com/windlant/protocol/types/skill_types"
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

	var tc *tools.AggregatedToolClient

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

	// 创建 Skill Client（从配置加载 LLM-driven skills）
	skillClient := skills.NewIntegratedClient(cfg)

	// 显示技能信息
	skillNames := skillClient.ListSkills()
	if len(skillNames) > 0 {
		fmt.Printf("✓ 已加载 %d 个本地技能\n", len(skillNames))
	} else {
		fmt.Println("⚠ 未加载任何本地技能")
	}

	a := agent.NewAgent(m, cfg.Context.MaxHistory, cfg.Tools.Enabled, tc, skillClient)

	// 注册 LLM skill executor：executor 接受 ctx，可从中读取绑定的 Agent（skills.ContextAgentKey）
	skillClient.RegisterLLMSkillExecutor(func(ctx context.Context, skillDef skill_types.SkillDefinition, input skill_types.SkillArguments) (interface{}, error) {
		// 读取 skill spec 文件
		specPath := fmt.Sprintf("internal/skills/manage/builtin/llm_driven_skills/specs/%s.md", skillDef.Name)
		specBytes, err := os.ReadFile(specPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read skill spec: %w", err)
		}
		specContent := string(specBytes)

		// 构建 prompt：包含 skill 说明书和参数信息
		prompt := fmt.Sprintf("执行技能：%s\n\n技能说明书：\n%s\n", skillDef.Name, specContent)

		// 如果有参数，说明参数信息
		if len(input) > 0 {
			prompt += "\n本次调用参数：\n"
			for k, v := range input {
				prompt += fmt.Sprintf("- %s: %v\n", k, v)
			}
		}

		// 优先从 ctx 中读取绑定的 Agent（TaskManager 会将 per-task Agent 放入 ctx）
		if ctx == nil {
			ctx = context.Background()
		}
		if val := ctx.Value(skills.ContextAgentKey); val != nil {
			if execAgent, ok := val.(*agent.Agent); execAgent != a && ok {
				return execAgent.ChatWithContext(ctx, prompt)
			}
		}

		// 默认使用主 agent（main 触发的调用）
		return prompt, nil
	})

	// 启动本地 A2A server（用于接收注册中心的 webhook 通知）
	registrar := a2aregistry.NewRegistrar()
	addr := fmt.Sprintf("%s:%d", cfg.Agent.A2A.Host, cfg.Agent.A2A.Port)
	// 提供一个 agentFactory 用于为每个远程 task 创建独立的 Agent
	agentFactory := func() *agent.Agent {
		return agent.NewAgent(m, cfg.Context.MaxHistory, cfg.Tools.Enabled, tc, skillClient)
	}
	if cfg.Agent.A2A.Enabled {
		_ = a2aserver.Start(addr, registrar, skillClient, agentFactory)
	}
	// 构建本地 AgentCard
	baseAgentURL := cfg.Agent.A2A.Host
	if !strings.HasPrefix(baseAgentURL, "http") {
		baseAgentURL = "http://" + baseAgentURL
	}
	baseAgentURL = strings.TrimRight(baseAgentURL, "/") + ":" + strconv.Itoa(cfg.Agent.A2A.Port)

	card := agent_types.AgentCard{
		AgentID:     cfg.Agent.ID,
		Name:        cfg.Agent.Name,
		Description: cfg.Agent.Description,
		Skills:      cfg.ConvertToSkillDefinitions(),
		URL:         baseAgentURL,
	}

	// 向注册中心注册（如果配置了 registry）
	var regClient *a2aclient.RegistryClient
	if cfg.Registry.RegistryServer.Host != "" {
		regBase := strings.TrimRight(cfg.Registry.RegistryServer.Host, "/") + ":" + strconv.Itoa(cfg.Registry.RegistryServer.Port)
		regClient = a2aclient.NewRegistryClient(regBase, cfg.Agent.ID)

		// 把本地 card 加入本地缓存，便于健康检查返回
		registrar.Update(card)

		webhookURL := baseAgentURL + "/registry/webhook"
		ctx := context.Background()
		resp, err := regClient.Register(ctx, card, webhookURL)
		if err != nil {
			log.Printf("注册到注册中心失败: %v", err)
		} else {
			log.Printf("已注册到注册中心，agentId=%s expiresAt=%s", resp.AgentID, resp.ExpiresAt)
		}

		// 在退出时注销
		defer func() {
			if regClient != nil {
				ctx := context.Background()
				if err := regClient.Deregister(ctx, cfg.Agent.ID); err != nil {
					log.Printf("注销到注册中心失败: %v", err)
				} else {
					log.Printf("已从注册中心注销: %s", cfg.Agent.ID)
				}
			}
		}()

		// 拉取所有 agent 列表并更新本地缓存
		agents, err := regClient.ListAgents(ctx)
		if err != nil {
			log.Printf("获取 agent 列表失败: %v", err)
		} else {
			for _, ac := range agents {
				if ac.AgentID == cfg.Agent.ID {
					continue
				}
				registrar.Update(ac)
				// 注册远程 agent 的 skills
				skillClient.RegisterRemoteAgentSkills(ac)
			}
			log.Printf("已同步 %d 个远程 agent 到本地缓存", len(agents))
		}

		// 设置远程 skill 调用函数
		a2aClient := a2aclient.NewA2AClient()
		skillClient.RegisterRemoteSkillCaller(func(ctx context.Context, targetAgent agent_types.AgentCard, skillID string, input skill_types.SkillArguments) (interface{}, error) {
			return a2aClient.CreateTask(ctx, targetAgent, skillID, input)
		})
	}

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
