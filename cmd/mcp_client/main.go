// cmd/mcp_client/main.go
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
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		fmt.Fprintln(os.Stderr, "Please ensure 'config/config.yaml' exists.")
		os.Exit(1)
	}

	m, err := model.NewDeepSeekModel(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize model: %v\n", err)
		os.Exit(1)
	}

	var tc tools.ToolClient

	if cfg.Tools.Enabled {
		switch cfg.Tools.Mode {
		case "local":
			tc = local.NewLocalToolClient()
			fmt.Println("Using local tool client (direct function calls).")

		case "stdio":
			serverBinary := "./cmd/mcp_server_local/mcp-server-local"
			tc, err = stdio.NewStdioToolClient(serverBinary)
			if err != nil {
				log.Fatalf("Failed to start stdio tool client: %v", err)
			}
			defer func() {
				if closeErr := tc.Close(); closeErr != nil {
					// log.Printf("Error closing stdio client: %v", closeErr)
				}
			}()
			// sigCh := make(chan os.Signal, 1)
			// signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
			// go func() {
			// 	<-sigCh
			// 	fmt.Println("\nShutting down MCP server...")
			// 	if err := tc.Close(); err != nil {
			// 		log.Printf("Warning: failed to close stdio client: %v", err)
			// 	}
			// 	os.Exit(0)
			// }()
			fmt.Println("Using stdio tool client (subprocess MCP server).")

		default:
			log.Fatalf("Unsupported tool mode: %s. Supported: local, stdio", cfg.Tools.Mode)
		}
	}

	a := agent.NewAgent(m, cfg.Context.MaxHistory, cfg.Tools.Enabled, tc)

	fmt.Println("MCP Client started!")
	if cfg.Tools.Enabled {
		fmt.Printf("Tool calling: enabled (mode: %s)\n", cfg.Tools.Mode)
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
