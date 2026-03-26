package main

import (
	"customclaw/internal/agent"
	"customclaw/internal/config"
	"customclaw/internal/llm"
	"customclaw/internal/tools"
	"errors"
	"fmt"
	"os"
)

// bootstrap loads config and actions, builds the LLM provider, tool registry,
// and agent. Shared by all commands.
func bootstrap(configPath, actionsPath string) (*config.Config, *config.Actions, *agent.Agent, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, nil, fmt.Errorf("%s not found — run './customclaw setup' to configure", configPath)
		}
		return nil, nil, nil, fmt.Errorf("load config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, nil, nil, fmt.Errorf("invalid config: %w", err)
	}

	actions, err := config.LoadActions(actionsPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("load actions: %w", err)
	}

	provider, err := buildProvider(cfg)
	if err != nil {
		return nil, nil, nil, err
	}

	registry, err := buildRegistryFromConfig(cfg, provider)
	if err != nil {
		return nil, nil, nil, err
	}

	ag := agent.New(provider, registry)
	return cfg, actions, ag, nil
}

// buildProvider constructs the LLM provider from config.
func buildProvider(cfg *config.Config) (llm.Provider, error) {
	switch cfg.LLM.Provider {
	case "anthropic":
		return llm.NewAnthropic(cfg.LLM.APIKey, cfg.LLM.Model), nil
	case "openai":
		return llm.NewOpenAI(cfg.LLM.APIKey, cfg.LLM.Model), nil
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s (supported: anthropic, openai)", cfg.LLM.Provider)
	}
}

// buildRegistry constructs the tool registry from config (for use in tools command).
func buildRegistry(configPath string) (*tools.Registry, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	provider, err := buildProvider(cfg)
	if err != nil {
		return nil, err
	}
	return buildRegistryFromConfig(cfg, provider)
}

// buildRegistryFromConfig registers all available tools based on the integration config.
func buildRegistryFromConfig(cfg *config.Config, provider llm.Provider) (*tools.Registry, error) {
	registry := tools.NewRegistry()

	registry.Register(tools.NewNotifyDiscord(cfg.Integrations.Discord.WebhookURL))
	registry.Register(tools.NewNotifyGoogleChat(cfg.Integrations.GoogleChat.WebhookURL))
	registry.Register(tools.NewGitHubCreateBranch(cfg.Integrations.GitHub.Token))
	registry.Register(tools.NewGitHubCreateIssue(cfg.Integrations.GitHub.Token))
	registry.Register(tools.NewGitHubCreateMR(cfg.Integrations.GitHub.Token))
	registry.Register(tools.NewJiraGetTicket(
		cfg.Integrations.Jira.BaseURL,
		cfg.Integrations.Jira.User,
		cfg.Integrations.Jira.APIToken,
	))
	registry.Register(tools.NewLLMCheckDescription(provider))

	return registry, nil
}
