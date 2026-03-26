package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Server       ServerConfig       `json:"server"`
	LLM          LLMConfig          `json:"llm"`
	Integrations IntegrationsConfig `json:"integrations"`
}

type ServerConfig struct {
	Port int `json:"port"`
}

type LLMConfig struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	APIKey   string `json:"api_key"`
}

type IntegrationsConfig struct {
	Discord    DiscordConfig    `json:"discord"`
	GoogleChat GoogleChatConfig `json:"google_chat"`
	GitHub     GitHubConfig     `json:"github"`
	Jira       JiraConfig       `json:"jira"`
}

type DiscordConfig struct {
	WebhookURL string `json:"webhook_url"`
}

type GoogleChatConfig struct {
	WebhookURL string `json:"webhook_url"`
}

type GitHubConfig struct {
	Token string `json:"token"`
}

type JiraConfig struct {
	BaseURL       string `json:"base_url"`
	WebhookSecret string `json:"webhook_secret"`
	User          string `json:"user"`
	APIToken      string `json:"api_token"`
}

type Actions struct {
	Tools     []string   `json:"tools"`
	Workflows []Workflow `json:"workflows"`
}

type Workflow struct {
	Name    string          `json:"name"`
	Trigger WorkflowTrigger `json:"trigger"`
	Goal    string          `json:"goal"`
}

type WorkflowTrigger struct {
	Type    string `json:"type"`
	Service string `json:"service"`
	Event   string `json:"event"`
	Path    string `json:"path"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	return &cfg, nil
}

func LoadActions(path string) (*Actions, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read actions: %w", err)
	}
	var actions Actions
	if err := json.Unmarshal(data, &actions); err != nil {
		return nil, fmt.Errorf("parse actions: %w", err)
	}
	return &actions, nil
}

func (c *Config) Validate() error {
	if c.LLM.Provider == "" {
		return fmt.Errorf("llm.provider is required")
	}
	if c.LLM.Model == "" {
		return fmt.Errorf("llm.model is required")
	}
	if c.LLM.APIKey == "" {
		return fmt.Errorf("llm.api_key is required")
	}
	return nil
}
