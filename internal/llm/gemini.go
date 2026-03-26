package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const geminiAPIURL = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s"
const geminiModelsURL = "https://generativelanguage.googleapis.com/v1beta/models?key=%s"

type GeminiProvider struct {
	apiKey string
	model  string
}

func NewGemini(apiKey, model string) *GeminiProvider {
	return &GeminiProvider{apiKey: apiKey, model: model}
}

func (p *GeminiProvider) Name() string { return "gemini" }

func (p *GeminiProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition) (*Response, error) {
	reqBody := map[string]any{
		"contents": p.buildContents(messages),
	}
	if len(tools) > 0 {
		reqBody["tools"] = []map[string]any{
			{"functionDeclarations": p.buildFunctionDeclarations(tools)},
		}
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf(geminiAPIURL, p.model, p.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("content-type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text         string `json:"text"`
					FunctionCall *struct {
						Name string         `json:"name"`
						Args map[string]any `json:"args"`
					} `json:"functionCall"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("gemini API error: %s", result.Error.Message)
	}
	if len(result.Candidates) == 0 {
		return nil, fmt.Errorf("empty response from gemini")
	}

	out := &Response{}
	for i, part := range result.Candidates[0].Content.Parts {
		if part.FunctionCall != nil {
			out.ToolCalls = append(out.ToolCalls, ToolCall{
				ID:    fmt.Sprintf("call_%d", i),
				Name:  part.FunctionCall.Name,
				Input: part.FunctionCall.Args,
			})
		} else {
			out.Content += part.Text
		}
	}
	return out, nil
}

func (p *GeminiProvider) buildContents(messages []Message) []map[string]any {
	out := make([]map[string]any, 0, len(messages))
	for _, m := range messages {
		switch m.Role {
		case "assistant":
			parts := []map[string]any{}
			if m.Content != "" {
				parts = append(parts, map[string]any{"text": m.Content})
			}
			for _, tc := range m.ToolCalls {
				parts = append(parts, map[string]any{
					"functionCall": map[string]any{
						"name": tc.Name,
						"args": tc.Input,
					},
				})
			}
			out = append(out, map[string]any{"role": "model", "parts": parts})
		case "tool":
			// Gemini expects function responses as user-role content
			out = append(out, map[string]any{
				"role": "user",
				"parts": []map[string]any{{
					"functionResponse": map[string]any{
						"name":     m.ToolCallID,
						"response": map[string]any{"content": m.Content},
					},
				}},
			})
		default:
			out = append(out, map[string]any{
				"role":  "user",
				"parts": []map[string]any{{"text": m.Content}},
			})
		}
	}
	return out
}

func (p *GeminiProvider) buildFunctionDeclarations(tools []ToolDefinition) []map[string]any {
	out := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		// Gemini expects uppercase type strings in parameter schemas.
		params := map[string]any{
			"type":       "OBJECT",
			"properties": buildGeminiProperties(t.Parameters.Properties),
		}
		if len(t.Parameters.Required) > 0 {
			params["required"] = t.Parameters.Required
		}
		out = append(out, map[string]any{
			"name":        t.Name,
			"description": t.Description,
			"parameters":  params,
		})
	}
	return out
}

func buildGeminiProperties(props map[string]PropertySchema) map[string]any {
	out := make(map[string]any, len(props))
	for name, p := range props {
		out[name] = map[string]any{
			"type":        strings.ToUpper(p.Type),
			"description": p.Description,
		}
	}
	return out
}

// ListGeminiModels fetches models that support generateContent from the Gemini API.
func ListGeminiModels(ctx context.Context, apiKey string) ([]string, error) {
	url := fmt.Sprintf(geminiModelsURL, apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Models []struct {
			Name                       string   `json:"name"`
			SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
		} `json:"models"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, fmt.Errorf("%s", result.Error.Message)
	}

	var models []string
	for _, m := range result.Models {
		for _, method := range m.SupportedGenerationMethods {
			if method == "generateContent" {
				// Strip "models/" prefix → "gemini-2.0-flash"
				name := strings.TrimPrefix(m.Name, "models/")
				models = append(models, name)
				break
			}
		}
	}
	return models, nil
}
