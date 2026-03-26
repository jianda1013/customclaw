package setup

import (
	"bufio"
	"context"
	"customclaw/internal/config"
	"customclaw/internal/llm"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"
)

// Wizard walks the user through an interactive configuration setup.
type Wizard struct {
	scanner *bufio.Scanner
	out     *os.File
}

func NewWizard() *Wizard {
	return &Wizard{
		scanner: bufio.NewScanner(os.Stdin),
		out:     os.Stdout,
	}
}

// Run executes the full setup wizard and writes config.json.
func (w *Wizard) Run(outputPath string) error {
	w.header("Welcome to customclaw!")
	fmt.Fprintln(w.out, "Let's configure your instance. Press Enter to accept defaults.")
	fmt.Fprintln(w.out)

	cfg := &config.Config{}

	if err := w.configureLLM(cfg); err != nil {
		return err
	}
	if err := w.configureServer(cfg); err != nil {
		return err
	}
	if err := w.configureIntegrations(cfg); err != nil {
		return err
	}

	if err := w.write(outputPath, cfg); err != nil {
		return err
	}

	fmt.Fprintln(w.out)
	fmt.Fprintf(w.out, "Configuration saved to %s\n", outputPath)
	fmt.Fprintln(w.out, "Run './customclaw validate' to verify, then './customclaw start' to begin.")
	return nil
}

func (w *Wizard) configureLLM(cfg *config.Config) error {
	w.section("LLM Configuration")

	provider := w.prompt("Provider", "anthropic", "anthropic, openai, gemini")
	cfg.LLM.Provider = provider

	apiKey := w.secret("API Key")
	if apiKey == "" {
		return fmt.Errorf("LLM API key is required")
	}
	cfg.LLM.APIKey = apiKey

	cfg.LLM.Model = w.selectModel(provider, apiKey)
	return nil
}

// selectModel fetches available models for the given provider and API key,
// then lets the user pick one from a numbered list.
// Falls back to a free-text prompt if the fetch fails.
func (w *Wizard) selectModel(provider, apiKey string) string {
	defaultModel := defaultModelFor(provider)

	fmt.Fprintf(w.out, "Fetching available models...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	models, err := fetchModels(ctx, provider, apiKey)
	if err != nil {
		fmt.Fprintf(w.out, " failed (%s)\n", err)
		return w.prompt("Model", defaultModel, "")
	}
	if len(models) == 0 {
		fmt.Fprintln(w.out, " no models returned.")
		return w.prompt("Model", defaultModel, "")
	}

	fmt.Fprintf(w.out, " found %d model(s)\n\n", len(models))

	// Find the index of the default model so we can highlight it.
	defaultIdx := -1
	for i, m := range models {
		marker := "  "
		if m == defaultModel {
			marker = "* "
			defaultIdx = i + 1
		}
		fmt.Fprintf(w.out, "  %s%2d. %s\n", marker, i+1, m)
	}
	fmt.Fprintln(w.out)

	hint := "enter number"
	if defaultIdx > 0 {
		hint = fmt.Sprintf("enter number, default %d", defaultIdx)
	}
	fmt.Fprintf(w.out, "Select model (%s): ", hint)

	if !w.scanner.Scan() {
		return defaultModel
	}
	input := strings.TrimSpace(w.scanner.Text())
	if input == "" {
		return defaultModel
	}

	// Accept a number (pick from list) or a raw model name (custom / unlisted).
	if n, err := strconv.Atoi(input); err == nil {
		if n >= 1 && n <= len(models) {
			return models[n-1]
		}
		fmt.Fprintf(w.out, "Invalid selection, using default (%s)\n", defaultModel)
		return defaultModel
	}
	return input // treat as a literal model name
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

func (w *Wizard) configureServer(cfg *config.Config) error {
	w.section("Server Configuration")
	portStr := w.prompt("Webhook server port", "8080", "")
	port := 8080
	fmt.Sscanf(portStr, "%d", &port)
	cfg.Server.Port = port
	return nil
}

func (w *Wizard) configureIntegrations(cfg *config.Config) error {
	w.section("Integrations")
	fmt.Fprintln(w.out, "Choose which integrations to enable.")
	fmt.Fprintln(w.out)

	if w.confirm("GitHub", true) {
		cfg.Integrations.GitHub.Token = w.secret("  GitHub personal access token")
	}

	fmt.Fprintln(w.out)
	if w.confirm("Jira", false) {
		cfg.Integrations.Jira.BaseURL = w.prompt("  Jira base URL", "https://your-org.atlassian.net", "")
		cfg.Integrations.Jira.User = w.prompt("  Jira user email", "", "")
		cfg.Integrations.Jira.APIToken = w.secret("  Jira API token")
		cfg.Integrations.Jira.WebhookSecret = w.secret("  Jira webhook secret (optional, press Enter to skip)")
	}

	fmt.Fprintln(w.out)
	if w.confirm("Discord", false) {
		cfg.Integrations.Discord.WebhookURL = w.prompt("  Discord webhook URL", "", "")
	}

	fmt.Fprintln(w.out)
	if w.confirm("Google Chat", false) {
		cfg.Integrations.GoogleChat.WebhookURL = w.prompt("  Google Chat webhook URL", "", "")
	}

	return nil
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

// prompt prints a labelled prompt with an optional default and hint, reads a line.
func (w *Wizard) prompt(label, defaultVal, hint string) string {
	display := label
	if hint != "" {
		display += " [" + hint + "]"
	}
	if defaultVal != "" {
		display += " (" + defaultVal + ")"
	}
	fmt.Fprintf(w.out, "%s: ", display)

	if !w.scanner.Scan() {
		return defaultVal
	}
	val := strings.TrimSpace(w.scanner.Text())
	if val == "" {
		return defaultVal
	}
	return val
}

// secret reads a line from the terminal without echoing characters.
func (w *Wizard) secret(label string) string {
	fmt.Fprintf(w.out, "%s: ", label)

	// Use raw terminal read if stdin is a TTY so characters are not echoed.
	if term.IsTerminal(int(os.Stdin.Fd())) {
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(w.out)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(b))
	}

	// Fallback for piped input (e.g. in tests).
	if !w.scanner.Scan() {
		return ""
	}
	return strings.TrimSpace(w.scanner.Text())
}

// confirm prints a yes/no prompt and returns true if the user answers y.
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
