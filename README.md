# Techne Code

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)]()

**Open source coding AI agent in Go — extensible, multi-provider, with sub-agent orchestration**

## What is Techne Code?

Techne Code is an AI-powered coding assistant built in Go. It's designed to be:

- **Extensible**: Plugin-based skill system for custom capabilities
- **Multi-provider**: Support for multiple LLM providers (OpenAI, Anthropic, local models)
- **Orchestrated**: Sub-agent architecture for complex tasks
- **Terminal-first**: Beautiful TUI built with Bubbletea

## Key Features

- 🤖 Multi-provider LLM support
- 🔧 Extensible skills system
- 🎯 Sub-agent orchestration
- 💻 Terminal UI (TUI)
- 🔒 Permission-based tool execution
- 📝 Context-aware code generation

## Getting Started

### Prerequisites

- Go 1.24 or higher
- An LLM provider API key (OpenAI, Anthropic, etc.)

### Installation

```bash
# Clone the repository
git clone https://github.com/Edcko/techne-code.git
cd techne-code

# Build
task build

# Run
./bin/techne
```

### Quick Start

```bash
# Set your API key
export OPENAI_API_KEY=your-key-here

# Run Techne Code
techne
```

## Architecture

```
techne-code/
├── cmd/techne/          # Application entrypoint
├── internal/
│   ├── agent/           # Core agent logic
│   ├── config/          # Configuration management
│   ├── db/              # Database/storage layer
│   ├── llm/             # LLM client abstractions
│   │   └── providers/   # Provider implementations
│   ├── permission/      # Permission system
│   ├── skills/          # Skill registry & loader
│   └── tools/           # Tool implementations
├── pkg/                 # Public packages
├── tui/                 # Terminal UI components
│   └── components/      # Reusable TUI components
├── docs/                # Documentation
└── scripts/             # Build & utility scripts
```

## Contributing

We welcome contributions! Please see our [Contributing Guide](docs/CONTRIBUTING.md) for details.

### Development

```bash
# Install development dependencies
task deps

# Run linter
task lint

# Run tests
task test

# Build
task build
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

Built with ❤️ using:
- [Bubbletea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Go](https://golang.org/) - The Go programming language
