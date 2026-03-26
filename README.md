# customclaw

A local AI agent service that automates workflows across external services (Jira, GitHub, GitLab, Discord, Google Chat, etc.) using LLM-driven decision making.

## How it works

customclaw supports two ways to trigger a workflow:

**1. Webhook workflow** — an external service (e.g. Jira) POSTs an event to customclaw. The LLM reads the event, decides which tools to invoke, and executes them.

**2. User command workflow** — you type a natural language command in the terminal. The LLM interprets the intent, picks the right tools, and carries out the actions.

Both modes share the same tool registry and agentic loop underneath.

```
Webhook (POST /webhook/jira)       CLI ("create branch for JIRA-123")
              \                                    /
               ──────────────┬────────────────────
                             |
                        Agent Loop
                   (LLM + tool calling)
                             |
                      Tool Registry
         [notify_discord]  [github_create_branch]
         [github_create_mr]  [jira_get_ticket]  ...
                             |
                   External Services
```

## Features

- **Interactive setup wizard** — guided first-run configuration with arrow-key model selection and current-value defaults on re-run
- **Multi-LLM support** — Anthropic, OpenAI, and Google Gemini; models fetched live from the provider API
- **LLM-driven tool selection** — define a goal, let the agent decide which tools to call and in what order
- **Webhook workflows** — trigger automations from Jira, GitHub, GitLab, ClickUp
- **CLI workflows** — one-shot `run` commands and an interactive `chat` REPL with command history
- **JSON-defined actions** — configure tools and workflows without writing code
- **Integrations** — GitHub, Jira, Discord, Google Chat

## Quick start

> Requires Go 1.22+

```bash
git clone https://github.com/jianda1013/customclaw.git
cd customclaw
go build -o customclaw ./cmd
./customclaw setup
```

## Setup wizard

Run `./customclaw setup` to configure everything interactively. Re-running it shows your current values as defaults — press Enter to keep any field unchanged.

```
──────────────────────────────────────────────────────
 Welcome to customclaw!
──────────────────────────────────────────────────────
Updating existing configuration. Press Enter to keep current values.

── LLM Configuration ───────────────────────────────────────────────
Provider [anthropic, openai, gemini] (anthropic):
API Key (currently: ...a3f2, press Enter to keep): ****

Fetching available models... found 8 model(s)

  ▶ claude-sonnet-4-6         ← ↑/↓ to navigate, Enter to select
    claude-opus-4-6
    claude-haiku-4-5-20251001
    ...

── Server Configuration ────────────────────────────────────────────
Webhook server port (8080):

── Integrations ────────────────────────────────────────────────────
GitHub [y/n] (y):
  GitHub personal access token (currently: ...kQ9x, press Enter to keep):
Jira [y/n] (n):
Discord [y/n] (n):
Google Chat [y/n] (n):

Configure workflows (actions.json) now [y/n] (n): y

── Tools ───────────────────────────────────────────────────────────
  [x] notify_discord          ← Space to toggle, Enter to confirm
  [x] github_create_branch
  [ ] github_create_issue
  ...

── Workflows ───────────────────────────────────────────────────────
  Workflow name: jira-ticket-created
  Trigger type: ▶ webhook
  Service:      ▶ jira
  Event (issue_created):
  Webhook path (/webhook/jira):
  Goal: Check description quality, notify Discord, create a branch
```

**Keyboard shortcuts in the wizard:**

| Key | Action |
|---|---|
| ↑ / ↓ | Navigate list |
| Enter | Confirm / accept default |
| Space | Toggle checkbox (tool selection) |
| Ctrl+C | Cancel — no changes saved |
| Ctrl+D | Same as Ctrl+C |

## Commands

```bash
# First-time (or update) configuration
./customclaw setup

# Start the webhook server
./customclaw start

# One-shot command
./customclaw run "check JIRA-123 and create a feature branch"

# Interactive REPL (↑/↓ for history, Ctrl+C to cancel a line, Ctrl+D to quit)
./customclaw chat

# Validate config.json and actions.json
./customclaw validate

# List all registered tools
./customclaw tools
```

Use `--config` and `--actions` to point to non-default files (useful for testing):

```bash
./customclaw --config /tmp/test.json --actions /tmp/test-actions.json setup
```

## Webhook server

```bash
./customclaw start
# customclaw listening on :8080
# registered webhook: POST /webhook/jira → workflow 'jira-ticket-created'
```

Configure your external service to POST events to `http://your-host:8080/webhook/jira`.

> For local development, expose the port with a tunnel: `ngrok http 8080`

## Configuration files

### config.json

```json
{
  "server": { "port": 8080 },
  "llm": {
    "provider": "anthropic",
    "model": "claude-sonnet-4-6",
    "api_key": "sk-ant-..."
  },
  "integrations": {
    "discord":     { "webhook_url": "https://discord.com/api/webhooks/..." },
    "google_chat": { "webhook_url": "https://chat.googleapis.com/..." },
    "github":      { "token": "ghp_..." },
    "jira": {
      "base_url":       "https://your-org.atlassian.net",
      "webhook_secret": "...",
      "user":           "you@example.com",
      "api_token":      "..."
    }
  }
}
```

Copy `config.example.json` as a starting point.

### actions.json

```json
{
  "tools": [
    "notify_discord",
    "notify_google_chat",
    "github_create_branch",
    "github_create_issue",
    "github_create_mr",
    "jira_get_ticket",
    "llm_check_description"
  ],
  "workflows": [
    {
      "name": "jira-ticket-created",
      "trigger": {
        "type": "webhook",
        "service": "jira",
        "event": "issue_created",
        "path": "/webhook/jira"
      },
      "goal": "A new Jira ticket was created. Check the description quality, notify the team on Discord, and create a feature branch and linked GitHub issue."
    }
  ]
}
```

Copy `actions.example.json` as a starting point.

## Supported LLM providers

| Provider | Default model | Notes |
|---|---|---|
| `anthropic` | `claude-sonnet-4-6` | Available models fetched live from `/v1/models` |
| `openai` | `gpt-4o` | Filtered to `gpt-*`, `o1*`, `o3*` models |
| `gemini` | `gemini-2.0-flash` | Filtered to models supporting `generateContent` |

## Available tools

| Tool | Description |
|---|---|
| `notify_discord` | Send a message to a Discord channel via webhook |
| `notify_google_chat` | Send a message to a Google Chat space |
| `github_create_branch` | Create a branch in a GitHub repo |
| `github_create_issue` | Create a GitHub issue |
| `github_create_mr` | Create a GitHub pull request |
| `jira_get_ticket` | Fetch ticket details from Jira |
| `llm_check_description` | Assess and improve a ticket or issue description |

## Testing

**Interactive:**
```bash
./customclaw --config /tmp/cfg.json --actions /tmp/act.json setup
./customclaw --config /tmp/cfg.json validate
./customclaw --config /tmp/cfg.json run "say hello"
```

**Scripted (pipe input):**
```bash
# Each line answers one setup prompt; non-TTY skips interactive selectors
printf 'anthropic\nsk-ant-KEY\n\n8080\nn\nn\nn\nn\nn\n' \
  | ./customclaw --config /tmp/cfg.json --actions /tmp/act.json setup
```

**Unit tests:**
```bash
go test ./...
```

## Project structure

```
customclaw/
├── cmd/
│   ├── main.go          # entry point
│   ├── root.go          # cobra root + persistent flags
│   ├── setup.go         # setup command
│   ├── start.go         # start command
│   ├── run.go           # run command
│   ├── chat.go          # chat command
│   ├── validate.go      # validate command
│   ├── tools.go         # tools command
│   └── bootstrap.go     # shared wiring: config → LLM → registry → agent
├── internal/
│   ├── config/          # Load config.json and actions.json
│   ├── agent/           # Agentic loop (ReAct pattern, max 20 iterations)
│   ├── llm/
│   │   ├── provider.go  # Provider interface + types
│   │   ├── anthropic.go # Anthropic implementation + ListAnthropicModels
│   │   ├── openai.go    # OpenAI implementation + ListOpenAIModels
│   │   └── gemini.go    # Gemini implementation + ListGeminiModels
│   ├── tools/
│   │   ├── tool.go      # Tool interface
│   │   ├── registry.go  # Registry: register, filter, execute
│   │   ├── notify.go    # Discord, Google Chat
│   │   ├── github.go    # Branch, issue, pull request
│   │   ├── jira.go      # Get ticket
│   │   └── llm_check.go # Description quality check
│   ├── setup/
│   │   ├── wizard.go        # Main setup wizard
│   │   ├── actions_wizard.go # Actions.json wizard (tools + workflows)
│   │   └── selector.go      # Arrow-key single-select and multi-select
│   └── triggers/
│       ├── webhook.go   # HTTP webhook server
│       └── cli.go       # run command + chat REPL
├── config.example.json
├── actions.example.json
└── CONTRIBUTING.md
```

## Roadmap

- [x] Phase 1: Core agent loop, webhook + CLI triggers, GitHub / Jira / Discord tools
- [x] Phase 1: Interactive setup wizard with live model fetching and arrow-key selection
- [x] Phase 1: Gemini LLM provider
- [ ] Phase 2: YAML action definitions, GitLab support, ClickUp support
- [ ] Phase 2: Mock LLM provider for testing
- [ ] Phase 3: Web dashboard, event history (SQLite)
