package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

// 启动 MCP 本地服务器，从标准输入逐行读取请求，处理后将响应写回标准输出
func main() {
	srv := NewServer()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Bytes()

		respBytes, err := srv.HandleRequest(line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			continue
		}

		_, err = os.Stdout.Write(respBytes)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Write error: %v\n", err)
			os.Exit(1)
		}

		// 添加换行符以符合 NDJSON 格式（每条 JSON 单独一行）
		_, err = os.Stdout.WriteString("\n")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Write error: %v\n", err)
			os.Exit(1)
		}
	}

	// 检查是否因非 EOF 原因导致读取失败
	if err := scanner.Err(); err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "Read error: %v\n", err)
		os.Exit(1)
	}
}
