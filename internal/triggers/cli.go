package triggers

import (
	"context"
	"customclaw/internal/agent"
	"customclaw/internal/config"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	result, err := c.agent.Run(ctx, command, c.actions.Tools, nil)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			fmt.Fprintln(os.Stdout, "\ncancelled.")
			return nil
		}
		return err
	}
	fmt.Println(result)
	return nil
}

// Chat starts an interactive REPL with readline support:
// - ←/→ move the cursor within the line
// - ↑/↓ scroll through command history (last 200 entries)
// - Ctrl+C at the prompt clears the current line
// - Ctrl+C during an agent run cancels it and returns to the prompt
// - Ctrl+D or 'exit' quits
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
			// Ctrl+C at the prompt — clear the line, stay in the loop.
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

		// Create a context that is cancelled when the user presses Ctrl+C.
		// This interrupts any in-flight HTTP call inside the agent.
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		result, err := c.agent.Run(ctx, line, c.actions.Tools, nil)
		stop() // always release the signal handler once the run is done

		if err != nil {
			if errors.Is(err, context.Canceled) {
				fmt.Fprintln(os.Stdout, "\ncancelled.")
				continue
			}
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			continue
		}
		fmt.Fprintln(os.Stdout, result)
		fmt.Fprintln(os.Stdout)
	}

	return nil
}
