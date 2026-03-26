package triggers

import (
	"context"
	"customclaw/internal/agent"
	"customclaw/internal/config"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/chzyer/readline"
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

// Chat starts an interactive REPL with readline support:
// arrow keys for cursor movement, ↑/↓ for history, Ctrl+C to cancel a line.
func (c *CLITrigger) Chat() error {
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "> ",
		HistoryLimit:    200,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return fmt.Errorf("init readline: %w", err)
	}
	defer rl.Close()

	fmt.Fprintln(os.Stdout, "customclaw chat — type your command, Ctrl+C to cancel, Ctrl+D or 'exit' to quit.")
	fmt.Fprintln(os.Stdout)

	for {
		line, err := rl.Readline()
		if err == readline.ErrInterrupt {
			// Ctrl+C clears the current line; loop continues.
			continue
		}
		if err == io.EOF {
			// Ctrl+D — exit cleanly.
			break
		}
		if err != nil {
			return err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			fmt.Fprintln(os.Stdout, "bye.")
			break
		}

		ctx := context.Background()
		result, err := c.agent.Run(ctx, line, c.actions.Tools, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			continue
		}
		fmt.Fprintln(os.Stdout, result)
		fmt.Fprintln(os.Stdout)
	}

	return nil
}
