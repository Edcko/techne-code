# Installing Techne

## Homebrew (macOS / Linux)

```bash
brew tap Edcko/tap
brew install techne
```

Upgrade to the latest version:

```bash
brew upgrade techne
```

## One-line Install (macOS / Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/Edcko/techne-code/main/install.sh | bash
```

The script detects your OS and architecture, downloads the latest release, verifies the checksum, and installs the binary to `/usr/local/bin/techne`.

It is safe to run multiple times — it will upgrade to the latest version.

## Go Install

Requires Go 1.22+.

```bash
go install github.com/Edcko/techne-code/cmd/techne@latest
```

The binary is installed to `$GOPATH/bin/techne` (or `$HOME/go/bin/techne` if `GOPATH` is unset).

## Build from Source

```bash
git clone https://github.com/Edcko/techne-code.git
cd techne-code
go build -o techne ./cmd/techne
./techne
```

For an optimized build:

```bash
go build -ldflags="-s -w" -o techne ./cmd/techne
```

## Verify Installation

```bash
techne version
```

## Shell Completion

Techne supports shell completions for bash, zsh, fish, and PowerShell:

```bash
techne completion bash > ~/.techne-completion.bash
techne completion zsh > ~/.techne-completion.zsh
techne completion fish > ~/.config/fish/completions/techne.fish
```
