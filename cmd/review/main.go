// Command review runs a code review from the terminal.
//
//	ANTHROPIC_API_KEY=... go run ./cmd/review path/to/file.go
//	cat snippet.js | ANTHROPIC_API_KEY=... go run ./cmd/review
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/asim/careem/reviewer"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	var code []byte
	var err error
	if len(os.Args) > 1 {
		code, err = os.ReadFile(os.Args[1])
	} else {
		code, err = io.ReadAll(os.Stdin)
	}
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(code)) == "" {
		return fmt.Errorf("no code provided: pass a file path or pipe code via stdin")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	result, err := reviewer.Review(ctx, string(code), reviewer.Options{})
	if err != nil {
		return err
	}
	fmt.Print(reviewer.ToMarkdown(result))
	return nil
}
