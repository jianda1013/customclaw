package tools

import (
	"context"
	"customclaw/internal/llm"
	"fmt"
)

// LLMCheckDescription uses the LLM to assess and improve a text description.
type LLMCheckDescription struct {
	provider llm.Provider
}

func NewLLMCheckDescription(provider llm.Provider) *LLMCheckDescription {
	return &LLMCheckDescription{provider: provider}
}

func (t *LLMCheckDescription) Name() string { return "llm_check_description" }

func (t *LLMCheckDescription) Description() string {
	return "Use the LLM to assess whether a ticket or issue description is clear and complete. Returns feedback and a suggested improved version."
}

func (t *LLMCheckDescription) Parameters() llm.ParameterSchema {
	return llm.ParameterSchema{
		Type: "object",
		Properties: map[string]llm.PropertySchema{
			"title":       {Type: "string", Description: "The ticket or issue title."},
			"description": {Type: "string", Description: "The ticket or issue description to evaluate."},
		},
		Required: []string{"title", "description"},
	}
}

func (t *LLMCheckDescription) Execute(ctx context.Context, input map[string]any) (string, error) {
	title, _ := input["title"].(string)
	description, _ := input["description"].(string)
	if title == "" || description == "" {
		return "", fmt.Errorf("title and description are required")
	}

	prompt := fmt.Sprintf(`You are reviewing a ticket description for clarity and completeness.

Title: %s
Description: %s

Assess whether this description is clear and actionable. Point out any missing information (acceptance criteria, steps to reproduce, expected vs actual behaviour, etc.). If the description is lacking, provide a concise improved version.

Reply in this format:
Assessment: <one short paragraph>
Suggested description: <improved version, or "No changes needed">`, title, description)

	resp, err := t.provider.Chat(ctx, []llm.Message{{Role: "user", Content: prompt}}, nil)
	if err != nil {
		return "", fmt.Errorf("llm check failed: %w", err)
	}
	return resp.Content, nil
}
