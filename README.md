# customclaw

A local AI agent service that automates workflows across external services (Jira, GitHub, GitLab, Discord, Google Chat, etc.) using LLM-driven decision making.

## How it works

customclaw supports two ways to trigger a workflow:

**1. Webhook workflow** — an external service (e.g. Jira) sends a webhook event to customclaw. The LLM reads the event, decides which tools to invoke, and executes them in sequence.

**2. User command workflow** — you type a natural language command in the terminal. The LLM interprets the intent, picks the right tools, and carries out the actions.

Both modes share the same tool registry and agentic loop underneath.

```
Webhook (POST /webhook/jira)          CLI ("create branch for JIRA-123")
              \                                      /
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

- LLM-driven tool selection — define a goal, let the agent decide which tools to call
- Pre-defined webhook workflows — trigger automations from Jira, GitHub, GitLab, ClickUp
- CLI interface — run one-shot commands or use the interactive REPL
- Multi-LLM support — bring your own API key; switch between OpenAI, Anthropic, and others
- JSON-defined actions — no code needed to define new workflows
- Integrations: GitHub, GitLab, Jira, Discord, Google Chat (more planned)

## Installation

> Requires Go 1.22+

```bash
git clone https://github.com/your-org/customclaw.git
cd customclaw
go build -o customclaw ./cmd
```

## Configuration

customclaw uses two config files in the project root.

### config.json

Global settings: LLM provider, API keys, server port, integration credentials.

```json
{
  "server": {
    "port": 8080
  },
  "llm": {
    "provider": "anthropic",
    "model": "claude-sonnet-4-6",
    "api_key": "sk-ant-..."
  },
  "integrations": {
    "discord": {
      "webhook_url": "https://discord.com/api/webhooks/..."
    },
    "google_chat": {
      "webhook_url": "https://chat.googleapis.com/..."
    },
    "github": {
      "token": "ghp_..."
    },
    "jira": {
      "base_url": "https://your-org.atlassian.net",
      "webhook_secret": "...",
      "user": "you@example.com",
      "api_token": "..."
    }
  }
}
```

### actions.json

Defines which tools are available and what webhook workflows exist.

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

## Usage

### Start the server (webhook mode)

```bash
./customclaw start
# Listening on :8080
# Webhook endpoint: POST /webhook/jira
```

Configure your external service (e.g. Jira) to send webhook events to `http://your-host:8080/webhook/jira`.

> For local development, use a tunnel to expose the port: `ngrok http 8080`

### Run a one-shot command

```bash
./customclaw run "check JIRA-123 and create a branch for it"
```

### Interactive REPL

```bash
./customclaw chat
> notify discord that JIRA-456 is ready for review
> also create a draft MR linked to that ticket
```

### Validate config

```bash
./customclaw validate
```

### List available tools

```bash
./customclaw tools
```

## Available tools

| Tool | Description |
|---|---|
| `notify_discord` | Send a message to a Discord channel via webhook |
| `notify_google_chat` | Send a message to a Google Chat space |
| `github_create_branch` | Create a branch in a GitHub repo |
| `github_create_issue` | Create a GitHub issue |
| `github_create_mr` | Create a GitHub pull request |
| `jira_get_ticket` | Fetch ticket details from Jira |
| `llm_check_description` | Use the LLM to assess and improve a text description |

## Supported LLM providers

| Provider | Models |
|---|---|
| Anthropic | claude-sonnet-4-6, claude-opus-4-6, claude-haiku-4-5 |
| OpenAI | gpt-4o, gpt-4o-mini, o1, o3-mini |

Set `llm.provider` and `llm.model` in `config.json` and provide the matching `api_key`.

## Project structure

```
customclaw/
├── cmd/
│   └── main.go               # CLI entrypoint
├── internal/
│   ├── config/               # Load and validate config.json / actions.json
│   ├── agent/                # Agentic loop: LLM + tool calling
│   ├── tools/                # Tool registry and implementations
│   │   ├── registry.go
│   │   ├── notify.go         # Discord, Google Chat
│   │   ├── github.go         # Branch, issue, MR
│   │   ├── jira.go           # Ticket fetch
│   │   └── llm_check.go      # Description quality check
│   ├── triggers/
│   │   ├── webhook.go        # HTTP server for webhook events
│   │   └── cli.go            # CLI command and REPL
│   └── llm/
│       ├── provider.go       # LLM interface
│       ├── anthropic.go
│       └── openai.go
├── config.json               # Your config (not committed)
├── actions.json              # Your workflow definitions
├── config.example.json       # Example config to copy from
├── actions.example.json      # Example actions to copy from
└── CONTRIBUTING.md
```

## Roadmap

- [ ] Phase 1: Core agent loop, webhook trigger, CLI trigger, GitHub + Jira + Discord tools
- [ ] Phase 2: YAML action definitions, GitLab support, ClickUp support, Google Chat
- [ ] Phase 3: Web dashboard, event history (SQLite), more LLM providers
