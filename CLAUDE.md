# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make build          # Build binary (output: ./app; set BINARY_NAME in Makefile if you want spec-tdd)
make test           # Run tests (race detector + coverage)
make lint           # Run golangci-lint
make vet            # Run go vet
make fmt            # Format code
make run            # Build and run
make docs           # Generate documentation (markdown)
make clean          # Clean build artifacts

# Single package test
go test -v -race ./internal/config/...
go test -v -race ./cmd/...

# Single test
go test -v -race -run TestFunctionName ./internal/config/...

# Devbox
devbox shell
devbox run build
devbox run test
```

## Architecture

```
main.go → cmd.Execute()
cmd/
  root.go         # Root command, Viper/Logger init, persistent flags (--config, --debug, --log-format)
  init.go         # spec-tdd init (workspace setup)
  req.go          # spec-tdd req add (REQ YAML creation)
  example.go      # spec-tdd example add (example append)
  map.go          # spec-tdd map (example mapping report)
  scaffold.go     # spec-tdd scaffold (test skeleton generation)
  trace.go        # spec-tdd trace (traceability report)
  version.go      # Version info (injected via ldflags: Version, Commit, BuildDate)
  completion.go   # Shell completion (bash/zsh/fish/powershell)
  docs.go         # Documentation generation (markdown/man/rest/yaml)
internal/
  apperrors/      # AppError type + sentinel errors (ErrNotFound, ErrInvalidInput, etc.)
  config/         # App config + spec config (.tdd/config.yml)
  logger/         # log/slog wrapper (text/json format, component tracking)
  spec/           # YAML DSL model + REQ/Example utilities
  scaffold/       # Test template rendering
  trace/          # Test scanning + report generation
```

**Config priority (app)**: CLI flags > env vars (`APP_` prefix) > config file (YAML) > defaults

**Spec config**: `.tdd/config.yml` (separate from app config)

**Version injection**: `make build` injects `cmd.Version`, `cmd.Commit`, `cmd.BuildDate` via ldflags.

## Key Patterns

- **Error handling**: `apperrors.Wrap("operation.name", err)` for contextual wrapping. `apperrors.IsNotFound(err)` for type checking.
- **Logging**: `cmd.GetLogger().WithComponent("name")` for component-scoped logger.
- **Testing**: Table-driven tests + `t.Run()` subtests. Capture output with `cmd.SetOut(&buf)` + `bytes.Buffer`.
- **Adding commands**: Create file in `cmd/`, call `rootCmd.AddCommand(newCmd)` in `init()`. Write output to `cmd.OutOrStdout()`.

## Spec-TDD Workflow (MVP)

```bash
spec-tdd init
spec-tdd req add --title "Login lockout after 5 failures"
spec-tdd example add --req REQ-001 --given "..." --when "..." --then "..."
spec-tdd scaffold
spec-tdd trace
spec-tdd map
```

Outputs:
- `.tdd/config.yml`, `.tdd/specs/*.yml`
- `tests/*.test.ts` (skeletons with TODO error)
- `.tdd/trace.json`, `.tdd/trace.md`, `.tdd/map.md`

## Gotchas

- Don't shadow stdlib package names (`errors`, `log`, etc.)
- Convert `slog.Attr` to `[]any` when passing to `slog.Logger.With()`
- Use `fmt.Fprintf(cmd.OutOrStdout(), ...)` instead of `fmt.Printf` for testability
- Run `go mod tidy` to keep direct/indirect markers correct
- `trace` only counts test titles that include `REQ-###` in `it("...")` or `test("...")`
- If `go test ./...` fails due to cache permissions on macOS, run with `GOCACHE=/tmp/go-build-cache`


# AI-DLC and Spec-Driven Development

Kiro-style Spec Driven Development implementation on AI-DLC (AI Development Life Cycle)

## Project Context

### Paths
- Steering: `.kiro/steering/`
- Specs: `.kiro/specs/`

### Steering vs Specification

**Steering** (`.kiro/steering/`) - Guide AI with project-wide rules and context
**Specs** (`.kiro/specs/`) - Formalize development process for individual features

### Active Specifications
- Check `.kiro/specs/` for active specifications
- Use `/kiro:spec-status [feature-name]` to check progress

## Development Guidelines
- Think in English, generate responses in Japanese. All Markdown content written to project files (e.g., requirements.md, design.md, tasks.md, research.md, validation reports) MUST be written in the target language configured for this specification (see spec.json.language).

## Minimal Workflow
- Phase 0 (optional): `/kiro:steering`, `/kiro:steering-custom`
- Phase 1 (Specification):
  - `/kiro:spec-init "description"`
  - `/kiro:spec-requirements {feature}`
  - `/kiro:validate-gap {feature}` (optional: for existing codebase)
  - `/kiro:spec-design {feature} [-y]`
  - `/kiro:validate-design {feature}` (optional: design review)
  - `/kiro:spec-tasks {feature} [-y]`
- Phase 2 (Implementation): `/kiro:spec-impl {feature} [tasks]`
  - `/kiro:validate-impl {feature}` (optional: after implementation)
- Progress check: `/kiro:spec-status {feature}` (use anytime)

## Development Rules
- 3-phase approval workflow: Requirements → Design → Tasks → Implementation
- Human review required each phase; use `-y` only for intentional fast-track
- Keep steering current and verify alignment with `/kiro:spec-status`
- Follow the user's instructions precisely, and within that scope act autonomously: gather the necessary context and complete the requested work end-to-end in this run, asking questions only when essential information is missing or the instructions are critically ambiguous.

## Steering Configuration
- Load entire `.kiro/steering/` as project memory
- Default files: `product.md`, `tech.md`, `structure.md`
- Custom files are supported (managed via `/kiro:steering-custom`)
