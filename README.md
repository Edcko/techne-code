<div align="center">

# Techne Code

**Open source coding AI agent in Go — extensible, multi-provider, with sub-agent orchestration**

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![CI](https://github.com/Edcko/techne-code/actions/workflows/ci.yml/badge.svg)](https://github.com/Edcko/techne-code/actions/workflows/ci.yml)
[![Release](https://github.com/Edcko/techne-code/actions/workflows/release.yml/badge.svg)](https://github.com/Edcko/techne-code/actions/workflows/release.yml)

[Installation](#installation) · [Quick Start](#quick-start) · [Configuration](#configuration) · [Architecture](#architecture) · [Contributing](CONTRIBUTING.md)

</div>

---

## What is Techne Code?

Techne Code is an AI-powered coding agent that runs in your terminal. Built in Go with a first-class TUI, it connects to multiple LLM providers and gives the AI a rich set of tools to read, write, search, and execute code — all with your approval.

What makes it different:

- **Multi-provider** — Anthropic Claude, OpenAI-compatible endpoints, Google Gemini, and local Ollama models
- **Sub-agent orchestration** — The agent delegates specialized tasks (research, coding, review, testing) to sub-agents that run in parallel
- **Extensible skills system** — Context-aware skill triggers inject domain expertise into the system prompt
- **Terminal-first** — Streaming TUI with markdown rendering, syntax highlighting, diff visualization, and multiline input
- **Permission-based** — Every destructive operation (file writes, shell commands) requires interactive approval

## Features

| Category | Details |
|----------|---------|
| **LLM Providers** | Anthropic Claude, OpenAI-compatible (Z.ai, OpenAI), Google Gemini, Ollama |
| **Per-provider tool control** | Ollama defaults to chat-only; cloud providers enable full agentic mode |
| **11 built-in tools** | `read_file`, `write_file`, `edit_file`, `glob`, `grep`, `bash`, `list_dir`, `git`, `web_fetch`, `delegate`, `subagent` |
| **8 built-in skills** | Go, TypeScript, Python, React, Docker, Database, API Design, Security |
| **Sub-agent orchestration** | 4 specialized agents (researcher, coder, reviewer, tester) with parallel delegation |
| **TUI** | Markdown rendering, syntax highlighting (7 languages), diff display, status bar, multiline input |
| **Slash commands** | `/model`, `/provider`, `/help`, `/clear` — switch models mid-session |
| **Session persistence** | SQLite-backed sessions with list, show, delete, and resume |
| **Context management** | Token tracking, automatic compression at 90% context window, LLM-powered summarization |
| **Permission system** | Interactive Y/A/N dialog for file modifications and shell commands |
| **Diagnostics** | `techne doctor` — 7 checks for config, providers, API keys, Ollama, and Go version |

## Installation

### Homebrew (macOS / Linux)

```bash
brew tap Edcko/homebrew-tap
brew install techne
```

### Curl (macOS / Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/Edcko/techne-code/main/install.sh | bash
```

### Go Install

```bash
go install github.com/Edcko/techne-code/cmd/techne@latest
```

### Build from Source

```bash
git clone https://github.com/Edcko/techne-code.git
cd techne-code
go build ./cmd/techne
```

> **Prerequisites**: Go 1.24+ and an LLM provider API key (or Ollama running locally).

## Quick Start

```bash
# Set your API key
export TECHNE_ANTHROPIC_API_KEY=sk-ant-...

# Launch interactive TUI
techne

# Or specify a provider and model
techne --provider openai --model glm-5

# Non-interactive (one-shot)
techne chat -p "Explain how Go interfaces work"

# Chat-only mode (no tools)
techne chat --no-tools -p "Tell me a joke about recursion"
```

## CLI Reference

### Global Flags

```
techne --provider <name>       # LLM provider (anthropic, openai, gemini, ollama)
techne --model <name>          # Model to use
techne --config <path>         # Path to config file
```

### Commands

```
techne                           # Interactive TUI (default)
techne chat                      # Interactive TUI (explicit)
techne chat -p "prompt"          # Non-interactive mode
techne chat --no-tools -p "Hi"   # Chat-only (no tool use)
techne chat --session <id>       # Resume a session
techne chat --new-session        # Force new session

techne session list              # List all sessions (alias: ls)
techne session show <id>         # Show session details and messages
techne session delete <id>       # Delete a session (alias: rm)

techne skills list               # List available skills with triggers
techne config init               # Bootstrap config interactively
techne config show               # Display effective config (masked keys)
techne doctor                    # Run diagnostic checks
techne version                   # Show version information
```

### TUI Slash Commands

| Command | Description |
|---------|-------------|
| `/model <name>` | Switch model mid-session |
| `/provider <name>` | Switch provider mid-session |
| `/clear` | Clear conversation history |
| `/help` | Show available commands |

**Keyboard shortcuts**: `Enter` = newline · `Ctrl+Enter` = send · `Ctrl+C` = quit/cancel

## Configuration

Techne Code looks for `techne.json` (or `techne.yaml`) in the current directory. You can bootstrap one with `techne config init`.

```json
{
  "default_provider": "anthropic",
  "default_model": "claude-sonnet-4-20250514",
  "providers": {
    "anthropic": {
      "type": "anthropic",
      "api_key": "${ANTHROPIC_API_KEY}",
      "models": ["claude-sonnet-4-20250514", "claude-3-5-sonnet-20241022"]
    },
    "openai": {
      "type": "openai",
      "api_key": "${OPENAI_API_KEY}",
      "base_url": "https://api.z.ai/api/coding/paas/v4",
      "models": ["glm-5", "glm-4.6", "glm-4.7"],
      "tools_enabled": true
    },
    "gemini": {
      "type": "gemini",
      "api_key": "${GEMINI_API_KEY}",
      "models": ["gemini-2.5-pro", "gemini-2.5-flash", "gemini-2.0-flash"],
      "tools_enabled": true
    },
    "ollama": {
      "type": "ollama",
      "base_url": "http://localhost:11434/v1",
      "models": ["llama3.2", "qwen2.5-coder", "deepseek-coder"],
      "tools_enabled": false
    }
  },
  "permissions": {
    "mode": "interactive",
    "allowed_tools": ["read_file", "glob", "grep"]
  },
  "skills": {
    "enabled": [],
    "disabled": [],
    "user_skills_path": "~/.config/techne/skills",
    "project_skills_path": ".techne/skills"
  },
  "options": {
    "context_paths": ["AGENTS.md", ".cursorrules"],
    "max_bash_timeout": 120000,
    "max_output_chars": 20000,
    "data_directory": ".techne/"
  }
}
```

### Configuration Reference

| Field | Description | Default |
|-------|-------------|---------|
| `default_provider` | Default LLM provider name | `"anthropic"` |
| `default_model` | Default model for the provider | Provider-specific |
| `providers.<name>.type` | Provider type: `anthropic`, `openai`, `gemini`, `ollama` | Required |
| `providers.<name>.api_key` | API key (supports `${VAR}` syntax) | From env |
| `providers.<name>.base_url` | Custom API endpoint | Provider-specific |
| `providers.<name>.models` | Available models list | `[]` |
| `providers.<name>.tools_enabled` | Enable tool use for this provider | `true` (except Ollama) |
| `permissions.mode` | `interactive`, `auto_allow`, `auto_deny` | `"interactive"` |
| `permissions.allowed_tools` | Tools that bypass permission check | `[]` |
| `skills.enabled` | Explicitly enabled skills | `[]` (all enabled) |
| `skills.disabled` | Explicitly disabled skills | `[]` |
| `options.context_paths` | Files to include in system prompt | `[]` |
| `options.max_bash_timeout` | Max bash command timeout (ms) | `120000` |
| `options.max_output_chars` | Max output before truncation | `20000` |
| `options.data_directory` | Data storage directory | `".techne/"` |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `TECHNE_ANTHROPIC_API_KEY` | API key for Anthropic Claude |
| `TECHNE_OPENAI_API_KEY` | API key for OpenAI-compatible providers |
| `TECHNE_GEMINI_API_KEY` | API key for Google Gemini |
| `TECHNE_OLLAMA_API_KEY` | API key for Ollama (optional) |

## Provider Setup

### Anthropic Claude

```bash
export TECHNE_ANTHROPIC_API_KEY=sk-ant-...
techne chat                                       # Default: claude-sonnet-4-20250514
```

Tools enabled by default. Supports streaming and tool calling.

### OpenAI-compatible (Z.ai, OpenAI)

```bash
export TECHNE_OPENAI_API_KEY=your-key-here
techne chat --provider openai --model glm-5
```

Works with any OpenAI-compatible endpoint. Configure `base_url` in the provider config. Tools enabled by default.

### Google Gemini

```bash
export TECHNE_GEMINI_API_KEY=your-key-here
techne chat --provider gemini --model gemini-2.5-pro
```

Supports models up to 1M context window (`gemini-2.5-pro`, `gemini-2.5-flash`, `gemini-2.0-flash`). Tool calling via REST API. Tools enabled by default.

### Ollama (Local Models)

```bash
# Start Ollama and pull a model
ollama serve
ollama pull qwen2.5-coder

# Chat-only mode (tools disabled by default for local models)
techne chat --provider ollama --model qwen2.5-coder -p "Explain Go interfaces"

# Force enable tools in config: "tools_enabled": true
```

Ollama runs locally — no API key required. Chat-only by default because many local models have limited tool-calling support. The adapter handles Ollama's non-standard tool call format transparently.

### Disabling Tools

```bash
techne chat --no-tools -p "Just a conversation, no file operations"
```

## Skills System

Skills inject domain-specific instructions into the system prompt based on context. They load from three locations:

- **Built-in**: Compiled into the binary
- **User skills**: `~/.config/techne/skills/`
- **Project skills**: `.techne/skills/`

### Built-in Skills

| Skill | Trigger | Description |
|-------|---------|-------------|
| `go_engineer` | `*.go` files | Go best practices, error handling, testing |
| `typescript` | `*.ts`, `*.tsx` files | TypeScript patterns and best practices |
| `python` | `*.py` files | Python idioms and best practices |
| `react` | `*.jsx`, `*.tsx` files | React component patterns |
| `docker` | `Dockerfile`, `docker-compose*` | Container best practices |
| `database` | `*.sql`, migration files | Database design and query patterns |
| `api_design` | API-related files | REST/GraphQL API design principles |
| `security` | Always active | Security-conscious coding practices |

### Custom Skills

Create a markdown file with YAML frontmatter in `~/.config/techne/skills/` or `.techne/skills/`:

```markdown
---
name: my-skill
description: Custom skill for my project
triggers:
  - type: file_pattern
    pattern: "*.py"
  - type: command
    pattern: "test"
---

## Instructions

Your custom instructions here. This content is injected into the system prompt
when the skill is active.

- Use markdown formatting
- Keep it focused and actionable
```

**Trigger types**: `always` (always active), `file_pattern` (glob match), `command` (command name match).

## Tools Reference

| Tool | Description | Permission |
|------|-------------|------------|
| `read_file` | Read file contents with optional line range | No |
| `write_file` | Create or overwrite files (auto-creates dirs) | Yes |
| `edit_file` | Find and replace text (exact match) | Yes |
| `glob` | Find files matching glob patterns | No |
| `grep` | Search file contents with regex (ripgrep fallback) | No |
| `bash` | Execute shell commands with timeout | Yes |
| `list_dir` | List directory contents with sizes and timestamps | No |
| `git` | Structured git commands (status, diff, log, add, commit, branch, stash) | Yes |
| `web_fetch` | HTTP GET with HTML-to-text conversion | No |
| `delegate` | Parallel sub-agent delegation (up to 3 concurrent tasks) | Yes |
| `subagent` | Execute a single sub-agent task (researcher, coder, reviewer, tester) | Yes |

### Sub-Agent Architecture

Four specialized agents are available through the `delegate` and `subagent` tools:

| Agent | Tools | Purpose |
|-------|-------|---------|
| `researcher` | `web_fetch`, `grep`, `glob` | Gather information and search code |
| `coder` | `write_file`, `edit_file`, `bash` | Implement changes and run builds |
| `reviewer` | `read_file`, `grep` | Review code for issues |
| `tester` | `bash` | Run tests and validate output |

The `delegate` tool accepts multiple tasks and runs up to 3 sub-agents concurrently, aggregating results.

## Session Management

Sessions are stored in a SQLite database (`.techne/techne.db`).

```bash
techne session list                # List all sessions
techne session show <id>           # Show details and message history
techne session delete <id>         # Delete a session

techne chat --session <id>         # Resume an existing session
techne chat --new-session          # Force start a new session
```

Sessions persist full conversation history across runs. The context manager tracks token usage and automatically compresses conversations when they approach the model's context window limit.

## Architecture

```
techne-code/
├── cmd/techne/                    # Application entrypoint
│   ├── main.go                    # Binary entry point
│   └── cli/                       # CLI commands (chat, session, skills, config, doctor, version)
├── internal/
│   ├── agent/                     # Core agent loop, sub-agent orchestration, context management
│   ├── config/                    # Configuration loading and validation (koanf-based)
│   ├── db/                        # SQLite database layer and session store
│   ├── event/                     # Channel-based event bus (sequential dispatch)
│   ├── llm/                       # LLM client abstractions
│   │   └── providers/
│   │       ├── anthropic/         # Anthropic Claude API adapter
│   │       ├── gemini/            # Google Gemini REST API adapter
│   │       ├── openai/            # OpenAI-compatible adapter (Z.ai, OpenAI)
│   │       └── ollama/            # Ollama local model adapter
│   ├── permission/                # Permission system (interactive/auto allow/deny)
│   ├── skills/                    # Skill registry, loader, prompt builder
│   │   └── builtin/               # 8 built-in skills
│   └── tools/                     # 11 tool implementations + sub-agent delegation
├── pkg/                           # Public packages (stable APIs)
│   ├── event/                     # Event types
│   ├── provider/                  # Provider interface
│   ├── session/                   # Session types
│   ├── skill/                     # Skill types and interfaces
│   └── tool/                      # Tool types and interfaces
├── tui/                           # Terminal UI (Bubbletea v2)
│   ├── components/                # Permission dialog, status bar
│   ├── diff/                      # LCS-based unified diff renderer
│   ├── markdown/                  # Markdown + syntax highlighting renderer
│   └── styles.go                  # Lipgloss styling
├── e2e/                           # End-to-end test suite
├── .github/workflows/             # CI (test/vet/build) + Release (GoReleaser)
├── docs/                          # Documentation
├── Formula/                       # Homebrew formula
├── install.sh                     # Curl-based installer
└── scripts/                       # Build and utility scripts
```

### Key Design Decisions

- **Sequential event bus** — Events dispatch through a channel to prevent race conditions in streaming output
- **Agent-as-Tool pattern** — Sub-agents implement the `tool.Tool` interface, so the LLM decides when to delegate
- **Provider adapters** — Each provider has dedicated `adapter.go` and `convert.go` files for clean separation of API quirks (e.g., Ollama's non-standard tool call format)
- **Permission events** — Agent and TUI communicate tool permission requests through the event bus, keeping concerns separated

## Development

### Running Tests

```bash
go test ./...                      # All tests
go test -v ./...                   # Verbose output
go test ./internal/tools/...       # Specific package
go test -count=1 ./...             # Disable test cache
```

### E2E Tests

The project includes end-to-end tests that run against a real Ollama instance:

```bash
# Requires Ollama running with qwen2.5-coder pulled
go test ./e2e/ -v -timeout 120s
```

### Code Quality

```bash
go vet ./...                       # Run linter
go fmt ./...                       # Format code
go mod verify                      # Verify dependencies
```

### Building

```bash
go build ./cmd/techne              # Local build
GOOS=linux GOARCH=amd64 go build ./cmd/techne
GOOS=darwin GOARCH=arm64 go build ./cmd/techne
GOOS=windows GOARCH=amd64 go build ./cmd/techne
```

### Test Coverage

| Package | What's Tested |
|---------|--------------|
| `cmd/techne/cli/` | CLI parsing, chat, config, doctor |
| `internal/agent/` | Agent loop, sub-agents, context management, token tracking |
| `internal/config/` | Config loading, defaults, validation |
| `internal/llm/providers/` | Anthropic (30 tests), Gemini (33 tests), OpenAI adapters |
| `internal/tools/` | All 11 tools — edit, glob, grep, bash, git, web_fetch, list_dir, delegate, subagent |
| `internal/skills/` | Registry, parser, loader |
| `tui/` | Model, slash commands, markdown rendering, diff, wrap, permission dialog |
| `e2e/` | Real Ollama chat, tool calling, multi-turn, session persistence, model switching |

## Contributing

Contributions are welcome! Check out the [Contributing Guide](CONTRIBUTING.md) for setup, code style, and PR process.

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes with tests
4. Run `go test ./...` and `go vet ./...`
5. Open a Pull Request

## License

[MIT](LICENSE)

## Built With

- [Bubbletea v2](https://charm.land/bubbletea/) — Terminal UI framework
- [Cobra](https://github.com/spf13/cobra) — CLI framework
- [Koanf](https://github.com/knadh/koanf) — Configuration management
- [Lipgloss](https://charm.land/lipgloss/) — Terminal styling
- [GoReleaser](https://goreleaser.ca/) — Release automation
