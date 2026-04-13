# Changelog

All notable changes to Techne Code will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Sub-agent orchestration with 4 built-in roles (researcher, coder, reviewer, tester)
- Google Gemini provider support
- Interactive permission dialog in TUI
- Context window management with LLM-powered summarization
- Markdown rendering with syntax highlighting in TUI
- Multiline input with paste support and input history
- `web_fetch` tool for URL content retrieval
- `git` tool for structured git operations
- `list_dir` tool for directory listing
- Diff display for file changes in TUI
- Status bar with model, tokens, and session info
- Session resume via `--session` flag
- `techne doctor` diagnostic command
- `techne config init/show` commands
- Ollama tool calling support with format conversion
- Per-provider `tools_enabled` configuration
- 5 new built-in skills: Python, React, Docker, Database, API Design
- Homebrew formula and install script
- GitHub Actions CI/CD pipeline
- 30+ Anthropic adapter mock tests

### Fixed

- Event bus race condition causing scrambled streaming output
- TUI space key not appending in chat input
- Session UUID display truncation
