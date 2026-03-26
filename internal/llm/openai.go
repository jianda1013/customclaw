package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const openaiAPIURL = "https://api.openai.com/v1/chat/completions"

type OpenAIProvider struct {
	apiKey string
	model  string
}

func NewOpenAI(apiKey, model string) *OpenAIProvider {
	return &OpenAIProvider{apiKey: apiKey, model: model}
}

func (p *OpenAIProvider) Name() string { return "openai" }

func (p *OpenAIProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition) (*Response, error) {
	reqBody := map[string]any{
		"model":    p.model,
		"messages": p.buildMessages(messages),
		"tools":    p.buildTools(tools),
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openaiAPIURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("content-type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("openai API error: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("empty response from openai")
	}

	msg := result.Choices[0].Message
	out := &Response{Content: msg.Content}
	for _, tc := range msg.ToolCalls {
		var input map[string]any
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &input); err != nil {
			return nil, fmt.Errorf("parse tool arguments: %w", err)
		}
		out.ToolCalls = append(out.ToolCalls, ToolCall{
			ID:    tc.ID,
			Name:  tc.Function.Name,
			Input: input,
		})
	}
	return out, nil
}

func (p *OpenAIProvider) buildMessages(messages []Message) []map[string]any {
	out := make([]map[string]any, 0, len(messages))
	for _, m := range messages {
		switch m.Role {
		case "assistant":
			msg := map[string]any{"role": "assistant", "content": m.Content}
			if len(m.ToolCalls) > 0 {
				tcs := make([]map[string]any, 0, len(m.ToolCalls))
				for _, tc := range m.ToolCalls {
					args, _ := json.Marshal(tc.Input)
					tcs = append(tcs, map[string]any{
						"id":   tc.ID,
						"type": "function",
						"function": map[string]any{
							"name":      tc.Name,
							"arguments": string(args),
						},
					})
				}
				msg["tool_calls"] = tcs
			}
			out = append(out, msg)
		case "tool":
			out = append(out, map[string]any{
				"role":         "tool",
				"tool_call_id": m.ToolCallID,
				"content":      m.Content,
			})
		default:
			out = append(out, map[string]any{"role": m.Role, "content": m.Content})
		}
	}
	return out
}

func (p *OpenAIProvider) buildTools(tools []ToolDefinition) []map[string]any {
	out := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		out = append(out, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.Parameters,
			},
		})
	}
	return out
}
