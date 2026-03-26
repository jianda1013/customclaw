package triggers

import (
	"bufio"
	"context"
	"customclaw/internal/agent"
	"customclaw/internal/config"
	"fmt"
	"os"
	"strings"
)

// CLITrigger handles user command workflows from the terminal.
type CLITrigger struct {
	actions *config.Actions
	agent   *agent.Agent
}

func NewCLITrigger(actions *config.Actions, ag *agent.Agent) *CLITrigger {
	return &CLITrigger{actions: actions, agent: ag}
}

// Run executes a single one-shot command.
func (c *CLITrigger) Run(command string) error {
	ctx := context.Background()
	result, err := c.agent.Run(ctx, command, c.actions.Tools, nil)
	if err != nil {
		return err
	}
	fmt.Println(result)
	return nil
}

// Chat starts an interactive REPL session.
func (c *CLITrigger) Chat() error {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("customclaw chat — type your command, or 'exit' to quit.")
	fmt.Println()

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			fmt.Println("bye.")
			break
		}

		ctx := context.Background()
		result, err := c.agent.Run(ctx, line, c.actions.Tools, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			continue
		}
		fmt.Println(result)
		fmt.Println()
	}

	return scanner.Err()
}
