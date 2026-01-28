package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/chzyer/readline"
	"github.com/windlant/mcp-client/internal/agent"
	"github.com/windlant/mcp-client/internal/config"
	"github.com/windlant/mcp-client/internal/model"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		fmt.Fprintln(os.Stderr, "Please ensure 'config/config.yaml' exists.")
		os.Exit(1)
	}

	m, err := model.NewDeepSeekModel(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize model: %v\n", err)
		os.Exit(1)
	}

	a := agent.NewAgent(m, cfg.Context.MaxHistory, cfg.Tools.Enabled)

	fmt.Println("MCP Client started!")
	if cfg.Tools.Enabled {
		fmt.Println("Tool calling: enabled")
	} else {
		fmt.Println("Tool calling: disabled")
	}
	fmt.Printf("Max context messages: %d\n", cfg.Context.MaxHistory)
	fmt.Println("Type 'exit' to quit, 'clear' to reset conversation history.")

	rl, err := readline.New("You: ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize input reader: %v\n", err)
		os.Exit(1)
	}
	defer rl.Close()

	// Main interaction loop
	for {
		line, err := rl.Readline()
		if err != nil {
			// Handle EOF (Ctrl+D) or read error
			break
		}

		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}

		// Handle built-in commands
		switch input {
		case "exit":
			fmt.Println("Goodbye!")
			return
		case "clear":
			a.ClearHistory()
			fmt.Println("Conversation history cleared.")
			continue
		}

		// Process user input with the agent
		reply, err := a.Chat(input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error processing request: %v\n", err)
			continue
		}

		fmt.Printf("Agent: %s\n\n", reply)
	}
}
