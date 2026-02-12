package builtin

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/chzyer/readline"
	"github.com/windlant/protocol/types/tools_types"
)

// AskTool 向指定目标提问并获取回答
// 参数:
// - content: 提问的内容
// - task_id: 提问对象，可以是具体的 task ID（向源 Agent 提问）或 "user"（向用户提问）
func AskTool(args tools_types.ToolArguments) (string, error) {
	// 验证必需参数
	content, ok := args["content"].(string)
	if !ok || content == "" {
		return "", fmt.Errorf("missing or invalid 'content' parameter")
	}

	taskID, ok := args["task_id"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'task_id' parameter")
	}

	// 处理向用户提问的情况
	if taskID == "user" {
		fmt.Printf("%s\n", content)

		// 创建 readline 实例，带提示符
		rl, err := readline.New("你的回答： ")
		if err != nil {
			// 如果 readline 初始化失败（如非交互式终端），回退到简单读取
			return fallbackReadLine(content)
		}
		defer rl.Close()

		// 读取一行输入（自动处理 \r\n、\n，且保证 UTF-8 完整性）
		answer, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				return "", fmt.Errorf("user interrupted input")
			}
			return "", fmt.Errorf("failed to read user input: %w", err)
		}

		// readline 已自动去除末尾换行符，无需手动 trim
		return answer, nil
	}

	// 处理向其他 Agent 提问的情况（保留接口）
	return fmt.Sprintf("[ASK_TOOL_PLACEHOLDER] Content: %s, TaskID: %s", content, taskID), nil
}
func fallbackReadLine(prompt string) (string, error) {
	fmt.Print("你的回答： ")
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	// 安全去除 \r\n 或 \n
	line = strings.TrimRight(line, "\r\n")
	return line, nil
}

var AskToolDef = tools_types.ToolDefinition{
	Name:        "ask_question",
	Description: "Ask a question to either the user or the source agent that initiated the current task. The task_id is provided by the system as context when the skill is invoked, allowing the model to reference the original requester. Use task_id='user' to request input from the human user, or use the provided task_id to communicate back to the source agent.",
	Parameters: tools_types.ToolSchema{
		Type: "object",
		Properties: map[string]tools_types.ToolParameter{
			"content": {
				Type:        "string",
				Description: "The question content to ask",
			},
			"task_id": {
				Type:        "string",
				Description: "The target to ask: 'user' for human input, or a specific task ID for agent-to-agent communication",
			},
		},
		Required: []string{"content", "task_id"},
	},
	Function: AskTool,
}
