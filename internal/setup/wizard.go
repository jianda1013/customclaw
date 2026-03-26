package setup

import (
	"context"
	"customclaw/internal/config"
	"customclaw/internal/llm"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chzyer/readline"
)

// ErrInterrupted is returned by Run when the user presses Ctrl+C or Ctrl+D.
var ErrInterrupted = errors.New("setup interrupted")

// Wizard walks the user through an interactive configuration setup.
type Wizard struct {
	rl  *readline.Instance
	out *os.File
	err error // sticky: set on first interrupt, short-circuits all later prompts
}

func NewWizard() (*Wizard, error) {
	rl, err := readline.NewEx(&readline.Config{
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return nil, fmt.Errorf("init readline: %w", err)
	}
	return &Wizard{rl: rl, out: os.Stdout}, nil
}

func (w *Wizard) Close() { w.rl.Close() }

// Run executes the full setup wizard and writes config.json.
// If the files already exist their values are loaded as defaults so the
// user can press Enter to keep any field unchanged.
// actionsPath is the path for actions.json; an empty string skips the actions step.
func (w *Wizard) Run(outputPath, actionsPath string) error {
	// Load existing config as the starting point (all fields become defaults).
	existing, err := config.Load(outputPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("load existing config: %w", err)
	}
	if existing == nil {
		existing = &config.Config{}
	}

	isUpdate := existing.LLM.APIKey != ""

	w.header("Welcome to customclaw!")
	if isUpdate {
		fmt.Fprintln(w.out, "Updating existing configuration. Press Enter to keep current values.")
	} else {
		fmt.Fprintln(w.out, "Let's configure your instance. Press Enter to accept defaults.")
	}
	fmt.Fprintln(w.out)

	cfg := *existing // copy — we'll overwrite field by field

	if err := w.configureLLM(&cfg); err != nil {
		return w.interruptedOrErr(err)
	}
	if err := w.configureServer(&cfg); err != nil {
		return w.interruptedOrErr(err)
	}
	if err := w.configureIntegrations(&cfg); err != nil {
		return w.interruptedOrErr(err)
	}

	if err := w.write(outputPath, &cfg); err != nil {
		return err
	}

	fmt.Fprintln(w.out)
	fmt.Fprintf(w.out, "Configuration saved to %s\n", outputPath)

	// Optional: configure actions.json now or later.
	if actionsPath != "" {
		fmt.Fprintln(w.out)
		if w.confirm("Configure workflows (actions.json) now", false) {
			if err := w.configureActions(actionsPath); err != nil {
				return err
			}
		} else {
			fmt.Fprintln(w.out, "You can configure workflows later by re-running './customclaw setup'.")
		}
	}

	fmt.Fprintln(w.out)
	fmt.Fprintln(w.out, "Run './customclaw validate' to verify, then './customclaw start' to begin.")
	return nil
}

func (w *Wizard) configureLLM(cfg *config.Config) error {
	w.section("LLM Configuration")

	providerDefault := cfg.LLM.Provider
	if providerDefault == "" {
		providerDefault = "anthropic"
	}
	cfg.LLM.Provider = w.prompt("Provider", providerDefault, "anthropic, openai, gemini")
	if w.err != nil {
		return w.err
	}

	apiKey := w.secret("API Key", cfg.LLM.APIKey)
	if w.err != nil {
		return w.err
	}
	if apiKey == "" {
		return fmt.Errorf("LLM API key is required")
	}
	cfg.LLM.APIKey = apiKey

	cfg.LLM.Model = w.selectModel(cfg.LLM.Provider, cfg.LLM.APIKey, cfg.LLM.Model)
	return w.err
}

func (w *Wizard) configureServer(cfg *config.Config) error {
	w.section("Server Configuration")
	port := cfg.Server.Port
	if port == 0 {
		port = 8080
	}
	portStr := w.prompt("Webhook server port", strconv.Itoa(port), "")
	if w.err != nil {
		return w.err
	}
	fmt.Sscanf(portStr, "%d", &cfg.Server.Port)
	return nil
}

func (w *Wizard) configureIntegrations(cfg *config.Config) error {
	w.section("Integrations")
	fmt.Fprintln(w.out, "Choose which integrations to enable.")
	fmt.Fprintln(w.out)

	if w.confirm("GitHub", cfg.Integrations.GitHub.Token != "") {
		cfg.Integrations.GitHub.Token = w.secret("  GitHub personal access token", cfg.Integrations.GitHub.Token)
	} else {
		cfg.Integrations.GitHub.Token = ""
	}
	if w.err != nil {
		return w.err
	}

	fmt.Fprintln(w.out)
	if w.confirm("Jira", cfg.Integrations.Jira.APIToken != "") {
		cfg.Integrations.Jira.BaseURL = w.prompt("  Jira base URL", orDefault(cfg.Integrations.Jira.BaseURL, "https://your-org.atlassian.net"), "")
		cfg.Integrations.Jira.User = w.prompt("  Jira user email", cfg.Integrations.Jira.User, "")
		cfg.Integrations.Jira.APIToken = w.secret("  Jira API token", cfg.Integrations.Jira.APIToken)
		cfg.Integrations.Jira.WebhookSecret = w.secret("  Jira webhook secret (optional, Enter to skip)", cfg.Integrations.Jira.WebhookSecret)
	} else {
		cfg.Integrations.Jira = config.JiraConfig{}
	}
	if w.err != nil {
		return w.err
	}

	fmt.Fprintln(w.out)
	if w.confirm("Discord", cfg.Integrations.Discord.WebhookURL != "") {
		cfg.Integrations.Discord.WebhookURL = w.prompt("  Discord webhook URL", cfg.Integrations.Discord.WebhookURL, "")
	} else {
		cfg.Integrations.Discord.WebhookURL = ""
	}
	if w.err != nil {
		return w.err
	}

	fmt.Fprintln(w.out)
	if w.confirm("Google Chat", cfg.Integrations.GoogleChat.WebhookURL != "") {
		cfg.Integrations.GoogleChat.WebhookURL = w.prompt("  Google Chat webhook URL", cfg.Integrations.GoogleChat.WebhookURL, "")
	} else {
		cfg.Integrations.GoogleChat.WebhookURL = ""
	}
	return w.err
}

// selectModel fetches available models and presents an interactive list.
// currentModel is pre-selected (marked with ▶ when the selector opens).
func (w *Wizard) selectModel(provider, apiKey, currentModel string) string {
	if w.err != nil {
		return orDefault(currentModel, defaultModelFor(provider))
	}

	fallbackDefault := orDefault(currentModel, defaultModelFor(provider))

	fmt.Fprintf(w.out, "Fetching available models...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	models, err := fetchModels(ctx, provider, apiKey)
	if err != nil {
		fmt.Fprintf(w.out, " failed (%s)\n", err)
		return w.prompt("Model", fallbackDefault, "")
	}
	if len(models) == 0 {
		fmt.Fprintln(w.out, " no models returned.")
		return w.prompt("Model", fallbackDefault, "")
	}

	fmt.Fprintf(w.out, " found %d model(s)\n\n", len(models))

	// Determine which index to pre-select:
	// current model first, then provider default, then index 0.
	defaultIdx := indexOfModel(models, currentModel)
	if defaultIdx < 0 {
		defaultIdx = indexOfModel(models, defaultModelFor(provider))
	}

	selected, err := interactiveSelect(models, defaultIdx)
	if err != nil {
		w.err = err
		return fallbackDefault
	}
	return selected
}

func (w *Wizard) write(path string, cfg *config.Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// prompt reads a line; defaultVal is shown and returned on empty input.
// If a previous interrupt was recorded it returns defaultVal immediately.
func (w *Wizard) prompt(label, defaultVal, hint string) string {
	if w.err != nil {
		return defaultVal
	}
	display := label
	if hint != "" {
		display += " [" + hint + "]"
	}
	if defaultVal != "" {
		display += " (" + defaultVal + ")"
	}

	w.rl.SetPrompt(display + ": ")
	line, err := w.rl.Readline()
	if err == readline.ErrInterrupt || err == io.EOF {
		fmt.Fprintln(w.out)
		w.err = ErrInterrupted
		return defaultVal
	}
	if err != nil {
		w.err = err
		return defaultVal
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultVal
	}
	return line
}

// secret reads masked input. If currentVal is set the prompt shows the last
// 4 characters so the user can confirm which key is stored, then returns
// currentVal on empty input.
// If a previous interrupt was recorded it returns currentVal immediately.
func (w *Wizard) secret(label, currentVal string) string {
	if w.err != nil {
		return currentVal
	}
	promptStr := label
	if currentVal != "" {
		promptStr += fmt.Sprintf(" (currently: ...%s, press Enter to keep)", last4(currentVal))
	}
	promptStr += ": "

	rl, err := readline.NewEx(&readline.Config{
		Prompt:     promptStr,
		EnableMask: true,
		MaskRune:   '*',
	})
	if err != nil {
		// Fallback: use the main readline instance without masking.
		w.rl.SetPrompt(promptStr)
		line, readErr := w.rl.Readline()
		if readErr == readline.ErrInterrupt || readErr == io.EOF {
			fmt.Fprintln(w.out)
			w.err = ErrInterrupted
			return currentVal
		}
		line = strings.TrimSpace(line)
		if line == "" {
			return currentVal
		}
		return line
	}
	defer rl.Close()

	line, err := rl.Readline()
	if err == readline.ErrInterrupt || err == io.EOF {
		fmt.Fprintln(w.out)
		w.err = ErrInterrupted
		return currentVal
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return currentVal
	}
	return line
}

// confirm prints a yes/no prompt; defaultYes controls the default answer.
func (w *Wizard) confirm(label string, defaultYes bool) bool {
	def := "n"
	if defaultYes {
		def = "y"
	}
	val := w.prompt(label, def, "y/n")
	return strings.ToLower(val) == "y"
}

func (w *Wizard) section(title string) {
	fmt.Fprintln(w.out)
	fmt.Fprintf(w.out, "── %s ", title)
	pad := 50 - len(title)
	if pad > 0 {
		fmt.Fprint(w.out, strings.Repeat("─", pad))
	}
	fmt.Fprintln(w.out)
}

func (w *Wizard) header(title string) {
	line := strings.Repeat("─", 54)
	fmt.Fprintln(w.out, line)
	fmt.Fprintln(w.out, " "+title)
	fmt.Fprintln(w.out, line)
}

// fetchModels dispatches to the right provider API.
func fetchModels(ctx context.Context, provider, apiKey string) ([]string, error) {
	switch provider {
	case "anthropic":
		return llm.ListAnthropicModels(ctx, apiKey)
	case "openai":
		return llm.ListOpenAIModels(ctx, apiKey)
	case "gemini":
		return llm.ListGeminiModels(ctx, apiKey)
	default:
		return nil, fmt.Errorf("unknown provider %q", provider)
	}
}

func defaultModelFor(provider string) string {
	switch provider {
	case "openai":
		return "gpt-4o"
	case "gemini":
		return "gemini-2.0-flash"
	default:
		return "claude-sonnet-4-6"
	}
}

func orDefault(val, fallback string) string {
	if val != "" {
		return val
	}
	return fallback
}

// indexOfModel returns the index of target in models, or -1 if not found.
func indexOfModel(models []string, target string) int {
	for i, m := range models {
		if m == target {
			return i
		}
	}
	return -1
}

// last4 returns the last 4 characters of s, or all of s if shorter.
func last4(s string) string {
	if len(s) <= 4 {
		return s
	}
	return s[len(s)-4:]
}

// interruptedOrErr prints a clean message for ErrInterrupted and returns it,
// or returns the original error for any other error type.
func (w *Wizard) interruptedOrErr(err error) error {
	if errors.Is(err, ErrInterrupted) {
		fmt.Fprintln(w.out, "\nSetup cancelled. No changes were saved.")
		return nil // exit 0, not an error
	}
	return err
}
