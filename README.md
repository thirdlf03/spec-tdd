# Cli template

A CLI application built with Go.

## Getting Started (Template)

This repository is a template for creating Go CLI applications. Use GitHub's "Use this template" button or clone manually:

```bash
git clone https://github.com/thirdlf03/spec-tdd.git my-app
cd my-app
chmod +x setup.sh
./setup.sh
```

`setup.sh` will interactively configure your project:

1. Replace module path and binary name
2. Update command metadata and description
3. Reset CHANGELOG.md
4. Run `go mod tidy`
5. Re-initialize git history
6. Self-delete after completion

## Installation

### Using `go install`

```bash
go install github.com/thirdlf03/spec-tdd@latest
```

### Build from Source

```bash
git clone https://github.com/thirdlf03/spec-tdd.git
cd go-cli-template
make build
```

## Usage

```bash
app --help
app version
app --debug
app --log-format json
app completion bash
app docs --format markdown --output ./docs
```

## Configuration

Configuration priority (highest to lowest):

1. Command-line flags
2. Environment variables (prefix: `APP_`)
3. Configuration file (`./config.yaml`, `./config/config.yaml`, `$HOME/config.yaml`)
4. Default values

```bash
app --config /path/to/config.yaml
```

## Development

### Using Devbox (Recommended)

```bash
devbox shell
devbox run build
devbox run test
devbox run lint
```

### Make Commands

```bash
make build    # Build binary
make test     # Run tests
make lint     # Run linter
make fmt      # Format code
make vet      # Run go vet
make run      # Build and run
make docs     # Generate documentation
make clean    # Clean build artifacts
make help     # Show all commands
```

## Project Structure

```
├── cmd/                   # Command implementations
│   ├── root.go            # Root command, Viper/Logger init
│   ├── version.go         # Version command
│   ├── completion.go      # Shell completion
│   └── docs.go            # Documentation generation
├── internal/
│   ├── apperrors/         # Error handling
│   ├── config/            # Configuration management
│   └── logger/            # Structured logging
├── config/
│   └── config.yaml.example
├── .github/
│   ├── workflows/         # GitHub Actions (CI, Release)
│   └── dependabot.yaml
├── .goreleaser.yaml
├── Makefile
├── main.go
└── go.mod
```

## License

MIT LICENSE [LICENSE](LICENSE) 
