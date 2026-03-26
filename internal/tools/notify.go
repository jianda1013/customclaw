package tools

import (
	"bytes"
	"context"
	"customclaw/internal/llm"
	"encoding/json"
	"fmt"
	"net/http"
)

// NotifyDiscord sends a message to a Discord channel via webhook.
type NotifyDiscord struct {
	webhookURL string
}

func NewNotifyDiscord(webhookURL string) *NotifyDiscord {
	return &NotifyDiscord{webhookURL: webhookURL}
}

func (t *NotifyDiscord) Name() string { return "notify_discord" }

func (t *NotifyDiscord) Description() string {
	return "Send a message to a Discord channel via webhook."
}

func (t *NotifyDiscord) Parameters() llm.ParameterSchema {
	return llm.ParameterSchema{
		Type: "object",
		Properties: map[string]llm.PropertySchema{
			"message": {Type: "string", Description: "The message content to send."},
		},
		Required: []string{"message"},
	}
}

func (t *NotifyDiscord) Execute(ctx context.Context, input map[string]any) (string, error) {
	message, _ := input["message"].(string)
	if message == "" {
		return "", fmt.Errorf("message is required")
	}
	body, _ := json.Marshal(map[string]string{"content": message})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.webhookURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("content-type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("discord returned status %d", resp.StatusCode)
	}
	return "Message sent to Discord.", nil
}

// NotifyGoogleChat sends a message to a Google Chat space via webhook.
type NotifyGoogleChat struct {
	webhookURL string
}

func NewNotifyGoogleChat(webhookURL string) *NotifyGoogleChat {
	return &NotifyGoogleChat{webhookURL: webhookURL}
}

func (t *NotifyGoogleChat) Name() string { return "notify_google_chat" }

func (t *NotifyGoogleChat) Description() string {
	return "Send a message to a Google Chat space via webhook."
}

func (t *NotifyGoogleChat) Parameters() llm.ParameterSchema {
	return llm.ParameterSchema{
		Type: "object",
		Properties: map[string]llm.PropertySchema{
			"message": {Type: "string", Description: "The message content to send."},
		},
		Required: []string{"message"},
	}
}

func (t *NotifyGoogleChat) Execute(ctx context.Context, input map[string]any) (string, error) {
	message, _ := input["message"].(string)
	if message == "" {
		return "", fmt.Errorf("message is required")
	}
	body, _ := json.Marshal(map[string]string{"text": message})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.webhookURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("content-type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("google chat returned status %d", resp.StatusCode)
	}
	return "Message sent to Google Chat.", nil
}
