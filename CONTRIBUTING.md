# Contributing to MCP Bridge

Thank you for your interest in contributing! This document covers development setup, code standards, and guidelines.

## Development Setup

```sh
# Clone
git clone https://github.com/jangtrinh/mcp-bridge-go.git
cd mcp-bridge-go

# Build
go build -o mcp-bridge .

# Run
MCP_BRIDGE_APP=Antigravity ./mcp-bridge
```

### Prerequisites

- Go 1.23+
- macOS (AppleScript dependency)
- Git

---

## Code Standards

### Go Conventions

This project follows idiomatic Go practices:

| Rule | Rationale |
|------|-----------|
| `gofmt` all code | Non-negotiable — run before every commit |
| `go vet` must pass | Catches common mistakes |
| `golangci-lint` must pass | Comprehensive linting |
| Short variable names | `cfg` not `configuration`, `s` not `server` in small scopes |
| Exported = PascalCase | `Config`, `LoadConfig` |
| Unexported = camelCase | `sendPrompt`, `gitCmd` |
| Error handling | Always check errors, wrap with `fmt.Errorf("context: %w", err)` |
| No blank error handling | Never `_ = someFunc()` for errors |

### Architecture Principles

| Principle | Application |
|-----------|-------------|
| **Single file** | Keep `main.go` as the single source file until complexity demands splitting |
| **No hardcoded paths** | All user-specific values via env vars (`MCP_BRIDGE_*`) |
| **Fail explicit** | Return clear error messages to MCP clients |
| **Minimal dependencies** | Only `mcp-go` SDK. No frameworks, no utility libs |
| **macOS-native** | Embrace AppleScript. Don't abstract away the platform |

### Security

| Rule | How |
|------|-----|
| No secrets in code | Use env vars, never commit tokens/keys |
| Validate MCP inputs | Check required params, validate paths |
| Annotate known-safe exec | Use `#nosec G204` with comment explaining why |
| Path safety | Use `filepath.Clean()` on user-provided paths |
| Truncate large outputs | Diff at 5KB, file previews at 500 chars |

### Quality Gates

Every PR must pass:

```sh
# 1. Format check
gofmt -d .

# 2. Static analysis
go vet ./...

# 3. Linting
golangci-lint run ./...

# 4. Security scan
gosec ./...

# 5. Build
go build -o /dev/null .
```

---

## Making Changes

### Single-File Rule

`main.go` is intentionally a single file (~300 lines). This makes the project:

- Easy to understand at a glance
- Simple to vendor or copy
- Quick to audit for security

**When to split:** Only when `main.go` exceeds ~500 lines or when adding a genuinely separate concern (e.g., a new transport layer).

### Adding New Tools

When adding a new MCP tool:

1. Define the tool with `mcp.NewTool()` — clear name, complete schema
2. Add handler function following the `handle*` naming pattern
3. Use existing helpers (`runOsascript`, `gitCmd`) where possible
4. Return structured text results with clear formatting
5. Update README with the new tool's API documentation

### Environment Variables

When adding new configuration:

1. Add to the `Config` struct with a clear field name
2. Load in `loadConfig()` with `envOr("MCP_BRIDGE_*", defaultValue)`
3. Document in README's Configuration table
4. Use sensible defaults so the tool works out of the box

---

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add support for Windsurf app
fix: handle empty clipboard gracefully
docs: add Paperclip integration example
refactor: extract AppleScript helpers
chore: update mcp-go to v0.30.0
```

## Pull Requests

- One concern per PR
- Include the "why" in the PR description
- Update docs if behavior changes
- All quality gates must pass

---

## Questions?

Open an issue on GitHub — we're happy to help.
