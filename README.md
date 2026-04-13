# Techne Code

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![CI](https://github.com/Edcko/techne-code/actions/workflows/ci.yml/badge.svg)](https://github.com/Edcko/techne-code/actions/workflows/ci.yml)

**Open source coding AI agent in Go — extensible, multi-provider, with sub-agent orchestration**

## What is Techne Code?

Techne Code is an AI-powered coding assistant built in Go. It's designed to be:

- **Extensible**: Plugin-based skill system for custom capabilities
- **Multi-provider**: Support for multiple LLM providers (Anthropic, OpenAI-compatible, Ollama)
- **Orchestrated**: Sub-agent architecture for complex tasks
- **Terminal-first**: Beautiful TUI built with Bubbletea v2

## Key Features

- **Multi-provider LLM support** — Anthropic, OpenAI-compatible (Z.ai, OpenAI), Ollama
- **Per-provider tools control** — Ollama defaults to chat-only, cloud providers enable full agentic mode
- **Extensible skills system** — Context-aware triggers for specialized coding patterns
- **Sub-agent orchestration** — Delegate complex tasks to specialized agents
- **Terminal UI with streaming** — Real-time thinking display, tool execution visualization
- **Session management** — Persistent conversation history with list/show/delete/resume
- **Permission-based execution** — Interactive approval for file modifications and shell commands
- **6 built-in tools** — read, write, edit, glob, grep, bash
- **3 built-in skills** — Go engineer, TypeScript, Security best practices

## Getting Started

### Prerequisites

- Go 1.24 or higher
- An LLM provider API key (Anthropic, OpenAI-compatible, or Ollama)

### Installation

```bash
# Clone the repository
git clone https://github.com/Edcko/techne-code.git
cd techne-code

# Build
go build ./cmd/techne

# Run
./techne
```

### Quick Start

```bash
# Set your API key (Anthropic)
export TECHNE_ANTHROPIC_API_KEY=your-key-here

# Or for OpenAI-compatible (Z.ai)
export TECHNE_OPENAI_API_KEY=your-key-here

# Run Techne Code (opens interactive TUI)
./techne

# Or specify provider/model
./techne --provider openai --model glm-5
```

## CLI Commands Reference

### Global Flags

```bash
techne --provider <name>       # LLM provider to use
techne --model <name>          # LLM model to use
techne --config <path>         # Path to config file
```

### Chat (Default Command)

```bash
techne                           # Interactive TUI (default)
techne chat                      # Interactive TUI (explicit)
techne chat -p "prompt"          # Non-interactive mode
techne chat --no-tools -p "Hi"   # Chat-only mode (no tool use)
techne --provider openai --model glm-5   # Use specific provider/model
```

### Session Management

```bash
techne session list              # List all sessions
techne session list              # Alias: techne session ls
techne session show <id>         # Show session details with messages
techne session delete <id>       # Delete a session (prompts for confirmation)
techne session delete <id>       # Alias: techne session rm <id>
```

### Skills

```bash
techne skills list               # List available skills with status and triggers
```

### Version

```bash
techne version                   # Show version information
```

## Configuration

### Config File

Techne Code looks for configuration in `techne.json` (or `techne.yaml`) in the current directory:

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
| `providers.<name>.type` | Provider type: `anthropic`, `openai`, `ollama` | Required |
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
| `TECHNE_OLLAMA_API_KEY` | API key for Ollama (optional) |

## Provider-Specific Notes

### Anthropic

- Set `TECHNE_ANTHROPIC_API_KEY` environment variable
- Default model: `claude-sonnet-4-20250514`
- Tools enabled by default

```bash
export TECHNE_ANTHROPIC_API_KEY=sk-ant-...
techne chat
```

### OpenAI-compatible (Z.ai, OpenAI)

- Set `TECHNE_OPENAI_API_KEY` environment variable
- For Z.ai: Base URL is `https://api.z.ai/api/coding/paas/v4`
- Available models: `glm-5`, `glm-4.6`, `glm-4.7`
- Tools enabled by default

```bash
export TECHNE_OPENAI_API_KEY=your-key-here
techne chat --provider openai --model glm-5
```

### Ollama (Local Models)

- Runs locally at `http://localhost:11434/v1`
- **No API key required**
- **Chat-only mode by default** — tools disabled for better compatibility with local models
- Force enable tools with `tools_enabled: true` in config

#### Using Ollama

1. **Install and start Ollama**:
   ```bash
   # Install Ollama (visit https://ollama.ai for instructions)
   # Start the Ollama service
   ollama serve
   ```

2. **Pull a model**:
   ```bash
   ollama pull qwen2.5-coder
   # Or other models: llama3.2, deepseek-coder, mistral
   ```

3. **Configure Techne Code**:
   ```json
   {
     "default_provider": "ollama",
     "default_model": "qwen2.5-coder",
     "providers": {
       "ollama": {
         "type": "ollama",
         "base_url": "http://localhost:11434/v1"
       }
     }
   }
   ```

4. **Run**:
   ```bash
   # Chat-only mode (default for Ollama)
   techne chat --provider ollama --model qwen2.5-coder -p "Explain Go interfaces"
   
   # Interactive session
   techne chat
   ```

### Disabling Tools

Use the `--no-tools` flag to force chat-only mode with any provider:

```bash
# Chat-only with OpenAI
techne chat --provider openai --model glm-5 --no-tools -p "Hello"

# Chat-only with Anthropic
techne chat --no-tools -p "Just chat, no file operations"
```

## Skills System

Skills provide specialized instructions to the AI based on context. They're loaded from:

- **Built-in**: Compiled into the binary
- **User skills**: `~/.config/techne/skills/`
- **Project skills**: `.techne/skills/`

### Built-in Skills

| Skill | Trigger | Description |
|-------|---------|-------------|
| `go_engineer` | `*.go` files | Go best practices, error handling, testing patterns |
| `typescript` | `*.ts`, `*.tsx` files | TypeScript patterns and best practices |
| `security` | Always active | Security-conscious coding practices |

### Creating Custom Skills

Skills are markdown files with YAML frontmatter:

```markdown
---
name: my-skill
description: Custom skill description
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
- Include practical guidelines
- Keep it focused and actionable
```

**Trigger Types**:

| Type | Pattern | Activation |
|------|---------|------------|
| `always` | (none) | Always active |
| `file_pattern` | `*.go`, `**/*.ts` | When editing matching files |
| `command` | `test`, `build` | When specific commands run |

## Tools Reference

| Tool | Description | Permission Required |
|------|-------------|---------------------|
| `read_file` | Read file contents with optional line range | No |
| `write_file` | Create or overwrite files | Yes |
| `edit_file` | Find and replace text in files | Yes |
| `glob` | Find files matching glob patterns | No |
| `grep` | Search file contents with regex | No |
| `bash` | Execute shell commands | Yes |

### Tool Details

#### read_file
```json
{"path": "main.go", "offset": 1, "limit": 50}
```
Returns file content with line numbers. Use offset/limit for large files.

#### write_file
```json
{"path": "output.txt", "content": "Hello, World!"}
```
Creates parent directories automatically.

#### edit_file
```json
{"path": "main.go", "old_string": "foo", "new_string": "bar"}
```
Exact match required. Returns number of replacements.

#### glob
```json
{"pattern": "**/*.go", "path": "src"}
```
Supports `**` for recursive matching. Results sorted alphabetically.

#### grep
```json
{"pattern": "func.*Handler", "include": "*.go", "path": "."}
```
Uses ripgrep if available, falls back to Go regex. Limits to 50 matches.

#### bash
```json
{"command": "go test ./...", "timeout_ms": 60000}
```
Executes in project directory. Output truncated at 20KB. Banned commands blocked.

## Session Management

Sessions store conversation history in a SQLite database (`.techne/techne.db`).

### Commands

```bash
# List all sessions
techne session list
# Output: ID  TITLE                    CREATED         MESSAGES
#         abc New Session              2025-01-15 10:30 5

# Show session details
techne session show abc
# Displays: ID, Title, Model, Provider, Created, Updated
# Plus message history (truncated to 500 chars per message)

# Delete a session (prompts for confirmation)
techne session delete abc
```

### Resuming Sessions

```bash
# Resume specific session
techne chat --session abc

# Force new session (don't resume)
techne chat --new-session
```

## Architecture

```
techne-code/
├── cmd/techne/              # Application entrypoint
│   └── cli/                 # CLI commands (chat, session, skills, version)
├── internal/
│   ├── agent/               # Core agent logic
│   ├── config/              # Configuration management (koanf-based)
│   ├── db/                  # SQLite database layer
│   ├── event/               # Channel-based event bus (sequential dispatch)
│   ├── llm/                 # LLM client abstractions
│   │   └── providers/       # Provider implementations
│   │       ├── anthropic/   # Anthropic Claude API
│   │       ├── openai/      # OpenAI-compatible (Z.ai, OpenAI)
│   │       └── ollama/      # Ollama local models
│   ├── permission/          # Permission system (interactive/auto)
│   ├── skills/              # Skill registry & loader
│   │   └── builtin/         # Built-in skills (go_engineer, typescript, security)
│   └── tools/               # Tool implementations (read, write, edit, glob, grep, bash)
├── pkg/                     # Public packages
│   ├── event/               # Event types (MessageDelta, ToolStart, ToolResult)
│   ├── provider/            # Provider interface
│   ├── session/             # Session types
│   ├── skill/               # Skill types and interfaces
│   └── tool/                # Tool types and interfaces
├── tui/                     # Terminal UI (Bubbletea v2)
│   └── components/          # Reusable TUI components
├── .github/workflows/       # GitHub Actions CI/CD
│   ├── ci.yml               # Test, vet, build on push/PR
│   └── release.yml          # Release automation
├── docs/                    # Documentation
└── scripts/                 # Build & utility scripts
```

### Event System

The event bus uses sequential dispatch to prevent race conditions:

```go
// Events are processed in order through a channel
bus := eventbus.NewChannelEventBus()
bus.Subscribe(func(e event.Event) {
    switch e.Type {
    case event.EventMessageDelta:
        // Handle streaming text
    case event.EventToolStart:
        // Tool execution started
    case event.EventToolResult:
        // Tool finished
    }
})
```

### TUI Features

- Real-time streaming with thinking display
- Tool execution visualization with status indicators
- Keyboard shortcuts: `ctrl+c` to quit/cancel, `enter` to send
- Responsive layout with message scrolling

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific package
go test ./internal/config/...
```

### Code Quality

```bash
# Run linter
go vet ./...

# Format code
go fmt ./...

# Verify dependencies
go mod verify
```

### Building

```bash
# Local build
go build ./cmd/techne

# Build for multiple platforms
GOOS=linux GOARCH=amd64 go build ./cmd/techne
GOOS=darwin GOARCH=arm64 go build ./cmd/techne
GOOS=windows GOARCH=amd64 go build ./cmd/techne
```

### Test Coverage

The project has comprehensive test coverage:

- `internal/config/` — Configuration loading, defaults, provider config
- `internal/tools/` — All 6 tools with various scenarios
- `internal/skills/` — Registry, parser, loader
- `cmd/techne/cli/` — CLI command parsing and execution

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Workflow

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests (`go test ./...`)
5. Run linter (`go vet ./...`)
6. Commit your changes
7. Push to the branch
8. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

Built with:
- [Bubbletea v2](https://charm.land/bubbletea/) — Terminal UI framework
- [Cobra](https://github.com/spf13/cobra) — CLI framework
- [Koanf](https://github.com/knadh/koanf) — Configuration management
- [Go](https://golang.org/) — The Go programming language
