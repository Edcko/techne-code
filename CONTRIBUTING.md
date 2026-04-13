# Contributing to Techne Code

Thanks for your interest in contributing. This document covers the basics.

## Reporting Bugs

Open a [GitHub issue](https://github.com/Edcko/techne-code/issues/new) with:

- Go version (`go version`)
- Techne version (`techne version`)
- Steps to reproduce
- Expected vs actual behavior

## Suggesting Features

Open a [GitHub issue](https://github.com/Edcko/techne-code/issues/new) with the `feature` label. Describe the problem you're solving and your proposed solution.

## Development Setup

### Prerequisites

- Go 1.24 or higher (see `go.mod`)
- Git

### Clone and Build

```bash
git clone https://github.com/Edcko/techne-code.git
cd techne-code
go build ./cmd/techne
```

### Run Tests

```bash
go test ./... -count=1
go vet ./...
```

### Run Locally

```bash
go run ./cmd/techne
```

## Code Style

We follow standard Go conventions with a few project-specific rules:

- **Vanilla Go testing** -- use `testing.T`, no testify or other test frameworks
- **Table-driven tests** -- prefer `[]struct{ want, got }` patterns
- **No unnecessary comments** -- code should be self-documenting; comment only when the "why" is not obvious
- **Formatting** -- `go fmt` before committing, no exceptions
- **Linter** -- `go vet` must pass clean

## Project Structure

```
cmd/techne/cli/        CLI commands (cobra)
internal/agent/        Core agent loop
internal/config/       Configuration (koanf)
internal/db/           SQLite persistence
internal/event/        Channel-based event bus
internal/llm/          LLM client + providers (anthropic, openai, ollama)
internal/permission/   Permission system
internal/skills/       Skill registry and loader
internal/tools/        Tool implementations (read, write, edit, glob, grep, bash)
pkg/                   Public types and interfaces
tui/                   Bubbletea v2 terminal UI
```

## Pull Request Process

1. Fork the repository
2. Create a branch from `main`:
   ```bash
   git checkout -b feature/short-description
   ```
3. Make your changes
4. Ensure tests and linter pass:
   ```bash
   go test ./... -count=1
   go vet ./...
   go fmt ./...
   ```
5. Commit with a descriptive message (see convention below)
6. Push and open a pull request against `main`

### CI Requirements

All PRs must pass the CI pipeline (`.github/workflows/ci.yml`):

- `go mod verify`
- `go vet ./...`
- `go test ./... -count=1`
- `go build ./...`

## Commit Messages

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add web_fetch tool for URL content retrieval
fix: resolve event bus race condition in streaming
docs: add installation guide for Homebrew
refactor: extract permission logic into separate package
test: add anthropic adapter mock tests
chore: update Go to 1.26.1
```

Prefixes: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `ci`, `perf`

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
