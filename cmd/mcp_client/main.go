// cmd/mcp_client/main.go
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/windlant/mcp-client/internal/agent"
	"github.com/windlant/mcp-client/internal/config"
	"github.com/windlant/mcp-client/internal/model"
)

func main() {
	// Load configuration from config/config.yaml
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		fmt.Fprintln(os.Stderr, "Please ensure 'config/config.yaml' exists.")
		os.Exit(1)
	}

	// Initialize the LLM backend
	m, err := model.NewDeepSeekModel(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize model: %v\n", err)
		os.Exit(1)
	}

	// Create an agent with context limit and tool usage enabled/disabled per config
	a := agent.NewAgent(m, cfg.Context.MaxHistory, cfg.Tools.Enabled)

	// Print startup info
	fmt.Println("MCP Client started!")
	if cfg.Tools.Enabled {
		fmt.Println("Tool calling: enabled")
	} else {
		fmt.Println("Tool calling: disabled")
	}
	fmt.Printf("Max context messages: %d\n", cfg.Context.MaxHistory)
	fmt.Println("Type 'exit' to quit, 'clear' to reset conversation history.")

	// Start interactive input loop
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break // Handle EOF (e.g., Ctrl+D)
		}

		input := strings.TrimSpace(scanner.Text())
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

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Input error: %v\n", err)
	}
}
