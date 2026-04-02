# Techne Code

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)]()

**Open source coding AI agent in Go — extensible, multi-provider, with sub-agent orchestration**

## What is Techne Code?

Techne Code is an AI-powered coding assistant built in Go. It's designed to be:

- **Extensible**: Plugin-based skill system for custom capabilities
- **Multi-provider**: Support for multiple LLM providers (Anthropic, OpenAI-compatible, Ollama)
- **Orchestrated**: Sub-agent architecture for complex tasks
- **Terminal-first**: Beautiful TUI built with Bubbletea

## Key Features

- Multi-provider LLM support (Anthropic, OpenAI-compatible/Z.ai, Ollama)
- Extensible skills system with context-aware triggers
- Sub-agent orchestration for complex workflows
- Terminal UI (TUI) with thinking display
- Session management with persistent history
- Permission-based tool execution
- Context-aware code generation

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

# Run Techne Code
techne
```

## Configuration

### Config File

Techne Code looks for configuration in `techne.json` (or `techne.yaml`):

```json
{
  "default_provider": "anthropic",
  "default_model": "claude-sonnet-4-20250514",
  "providers": {
    "anthropic": {
      "type": "anthropic",
      "api_key": "${ANTHROPIC_API_KEY}"
    },
    "openai": {
      "type": "openai",
      "api_key": "${OPENAI_API_KEY}",
      "base_url": "https://api.z.ai/api/coding/paas/v4"
    },
    "ollama": {
      "type": "ollama",
      "base_url": "http://localhost:11434/v1"
    }
  },
  "permissions": {
    "mode": "interactive"
  },
  "options": {
    "context_paths": ["AGENTS.md", ".cursorrules"],
    "max_bash_timeout": 120000,
    "max_output_chars": 20000,
    "data_directory": ".techne/"
  }
}
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `TECHNE_ANTHROPIC_API_KEY` | API key for Anthropic Claude |
| `TECHNE_OPENAI_API_KEY` | API key for OpenAI-compatible providers |
| `TECHNE_OLLAMA_API_KEY` | API key for Ollama (optional, uses "ollama" by default) |

### Skills Directory Structure

Skills are loaded from:
- User skills: `~/.config/techne/skills/`
- Project skills: `.techne/skills/`

## CLI Commands Reference

### Chat

```bash
techne                           # Interactive TUI (default)
techne chat                      # Interactive TUI (explicit)
techne chat -p "prompt"          # Non-interactive mode
techne --provider openai --model glm-5   # Use specific provider/model
```

### Session Management

```bash
techne session list              # List all sessions
techne session show <id>         # Show session details
techne session delete <id>       # Delete a session
```

### Skills

```bash
techne skills list               # List available skills
```

### Version

```bash
techne version                   # Show version
```

## Provider-Specific Notes

### Anthropic

- Set `TECHNE_ANTHROPIC_API_KEY` environment variable
- Default model: `claude-sonnet-4-20250514`

### OpenAI-compatible (Z.ai)

- Set `TECHNE_OPENAI_API_KEY` environment variable
- Base URL: `https://api.z.ai/api/coding/paas/v4`
- Available models: `glm-5`, `glm-4.6`, `glm-4.7`

### Ollama

- Runs locally at `http://localhost:11434/v1`
- No API key required (uses dummy value "ollama")
- Available models: `llama3`, `llama3.1`, `llama3.2`, `codellama`, `deepseek-coder`, `qwen2.5-coder`, `mistral`

## Architecture

```
techne-code/
├── cmd/techne/              # Application entrypoint
│   └── cli/                 # CLI commands (chat, session, skills)
├── internal/
│   ├── agent/               # Core agent logic
│   ├── config/              # Configuration management
│   ├── db/                  # Database/storage layer
│   ├── event/               # Event bus system
│   ├── llm/                 # LLM client abstractions
│   │   └── providers/       # Provider implementations
│   │       ├── anthropic/   # Anthropic Claude
│   │       ├── openai/      # OpenAI-compatible (Z.ai)
│   │       └── ollama/      # Ollama local models
│   ├── permission/          # Permission system
│   ├── skills/              # Skill registry & loader
│   │   └── builtin/         # Built-in skills
│   └── tools/               # Tool implementations
├── pkg/                     # Public packages
│   ├── event/               # Event types
│   ├── provider/            # Provider interface
│   ├── session/             # Session types
│   ├── skill/               # Skill types
│   └── tool/                # Tool types
├── tui/                     # Terminal UI components
│   └── components/          # Reusable TUI components
├── docs/                    # Documentation
└── scripts/                 # Build & utility scripts
```

## Contributing

We welcome contributions! Please see our [Contributing Guide](docs/CONTRIBUTING.md) for details.

### Development

```bash
# Run linter
go vet ./...

# Run tests
go test ./...

# Build
go build ./...
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

Built with:
- [Bubbletea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Go](https://golang.org/) - The Go programming language
