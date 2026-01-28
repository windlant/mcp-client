package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

func main() {
	srv := NewServer()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Bytes()

		respBytes, err := srv.HandleRequest(line)
		if err != nil {
			// Write error to stderr for debugging, but continue
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			continue
		}

		// Write response to stdout
		_, err = os.Stdout.Write(respBytes)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Write error: %v\n", err)
			os.Exit(1)
		}
		// Add newline for ndjson format
		_, err = os.Stdout.WriteString("\n")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Write error: %v\n", err)
			os.Exit(1)
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "Read error: %v\n", err)
		os.Exit(1)
	}
}
