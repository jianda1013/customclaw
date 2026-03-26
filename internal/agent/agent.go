package agent

import (
	"context"
	"customclaw/internal/llm"
	"customclaw/internal/tools"
	"fmt"
)

const maxIterations = 20

// Agent runs the agentic loop: sends a goal to the LLM, executes tool calls,
// feeds results back, and repeats until the LLM stops calling tools.
type Agent struct {
	llm      llm.Provider
	registry *tools.Registry
}

func New(provider llm.Provider, registry *tools.Registry) *Agent {
	return &Agent{llm: provider, registry: registry}
}

// Run executes a goal using only the specified tools (by name).
// Pass nil or empty slice to allow all registered tools.
// eventContext is optional key/value data included in the initial prompt.
func (a *Agent) Run(ctx context.Context, goal string, allowedTools []string, eventContext map[string]any) (string, error) {
	toolSet := a.registry.Filter(allowedTools)
	defs := tools.Definitions(toolSet)

	userMessage := buildPrompt(goal, eventContext)
	messages := []llm.Message{{Role: "user", Content: userMessage}}

	var lastContent string

	for i := 0; i < maxIterations; i++ {
		resp, err := a.llm.Chat(ctx, messages, defs)
		if err != nil {
			return "", fmt.Errorf("llm error on iteration %d: %w", i+1, err)
		}

		lastContent = resp.Content

		// No tool calls — LLM is done.
		if len(resp.ToolCalls) == 0 {
			return lastContent, nil
		}

		// Append assistant message (may include both text and tool calls).
		messages = append(messages, llm.Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		// Execute each tool call and append results.
		for _, tc := range resp.ToolCalls {
			result, err := a.registry.Execute(ctx, tc.Name, tc.Input)
			if err != nil {
				result = fmt.Sprintf("error: %s", err)
			}
			fmt.Printf("[tool] %s → %s\n", tc.Name, result)
			messages = append(messages, llm.Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
			})
		}
	}

	return "", fmt.Errorf("agent reached maximum iterations (%d) without completing", maxIterations)
}

func buildPrompt(goal string, ctx map[string]any) string {
	prompt := "Goal: " + goal
	if len(ctx) == 0 {
		return prompt
	}
	prompt += "\n\nContext:\n"
	for k, v := range ctx {
		prompt += fmt.Sprintf("- %s: %v\n", k, v)
	}
	return prompt
}
