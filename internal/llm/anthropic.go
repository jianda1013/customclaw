package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const anthropicAPIURL = "https://api.anthropic.com/v1/messages"

type AnthropicProvider struct {
	apiKey string
	model  string
}

func NewAnthropic(apiKey, model string) *AnthropicProvider {
	return &AnthropicProvider{apiKey: apiKey, model: model}
}

func (p *AnthropicProvider) Name() string { return "anthropic" }

func (p *AnthropicProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition) (*Response, error) {
	reqBody := map[string]any{
		"model":      p.model,
		"max_tokens": 4096,
		"messages":   p.buildMessages(messages),
		"tools":      p.buildTools(tools),
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicAPIURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Content []struct {
			Type  string         `json:"type"`
			Text  string         `json:"text"`
			ID    string         `json:"id"`
			Name  string         `json:"name"`
			Input map[string]any `json:"input"`
		} `json:"content"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("anthropic API error: %s", result.Error.Message)
	}

	out := &Response{}
	for _, block := range result.Content {
		switch block.Type {
		case "text":
			out.Content += block.Text
		case "tool_use":
			out.ToolCalls = append(out.ToolCalls, ToolCall{
				ID:    block.ID,
				Name:  block.Name,
				Input: block.Input,
			})
		}
	}
	return out, nil
}

func (p *AnthropicProvider) buildMessages(messages []Message) []map[string]any {
	out := make([]map[string]any, 0, len(messages))
	for _, m := range messages {
		switch m.Role {
		case "assistant":
			content := []map[string]any{{"type": "text", "text": m.Content}}
			for _, tc := range m.ToolCalls {
				content = append(content, map[string]any{
					"type":  "tool_use",
					"id":    tc.ID,
					"name":  tc.Name,
					"input": tc.Input,
				})
			}
			out = append(out, map[string]any{"role": "assistant", "content": content})
		case "tool":
			out = append(out, map[string]any{
				"role": "user",
				"content": []map[string]any{{
					"type":        "tool_result",
					"tool_use_id": m.ToolCallID,
					"content":     m.Content,
				}},
			})
		default:
			out = append(out, map[string]any{"role": m.Role, "content": m.Content})
		}
	}
	return out
}

func (p *AnthropicProvider) buildTools(tools []ToolDefinition) []map[string]any {
	out := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		out = append(out, map[string]any{
			"name":         t.Name,
			"description":  t.Description,
			"input_schema": t.Parameters,
		})
	}
	return out
}
