# MCP Bridge

> Lightweight MCP server that bridges AI coding tools via macOS automation.

**MCP Bridge** lets one AI agent delegate tasks to another by automating any Electron-based AI IDE (Antigravity, Cursor, Windsurf) through macOS AppleScript. It exposes two MCP tools over stdio — send a prompt and check workspace changes.

## Why

Modern AI-assisted development often involves **multiple AI tools**, each with different strengths. But they can't talk to each other natively. MCP Bridge solves this by:

- Giving any MCP-compatible AI agent (Gemini CLI, Claude Desktop, etc.) the ability to **delegate tasks** to another AI IDE
- **Monitoring results** by watching git status for file changes
- Working with **any Electron-based AI IDE** that has a keyboard shortcut to open a chat panel

---

## Use Cases

### 1. Gemini CLI → Antigravity Delegation

You're using Gemini CLI for orchestration but want Antigravity to handle complex code generation:

```
You (to Gemini CLI): "Refactor the authentication module to use JWT tokens"
  └── Gemini CLI calls `send_to_app` with the prompt
       └── MCP Bridge pastes the prompt into Antigravity's chat
            └── Antigravity processes the task, edits files
                 └── MCP Bridge detects git changes, returns diff to Gemini CLI
                      └── Gemini CLI reviews the changes and reports back to you
```

### 2. Multi-Agent Orchestration with Paperclip

Use [Paperclip](https://github.com/paperclipai/paperclip) to run AI agent companies where a CEO agent delegates coding tasks:

```
Paperclip CEO Agent
  └── Assigns task to CTO agent (Gemini CLI)
       └── CTO uses MCP Bridge to delegate to Antigravity
            └── Antigravity writes the code
                 └── CTO reviews via check_workspace_changes
                      └── Reports completion back to CEO
```

### 3. Cross-IDE Code Review

Use one AI tool to review another's output:

```
Claude Desktop
  └── Sends a coding task to Cursor via MCP Bridge
       └── Cursor generates the code
            └── Claude Desktop reads the diff
                 └── Claude reviews and provides feedback
```

### 4. Automated Task Pipelines

Chain multiple delegations in sequence — planning in one tool, implementation in another:

```
Orchestrator AI
  ├── Step 1: send_to_app → "Plan the database schema for a todo app"
  │    └── Reviews the plan from the diff
  ├── Step 2: send_to_app → "Implement the schema from the plan above"
  │    └── Verifies implementation matches plan
  └── Step 3: send_to_app → "Write tests for the database layer"
       └── Final verification
```

---

## How It Works

```
┌──────────────┐     MCP stdio      ┌─────────────┐     AppleScript     ┌──────────────┐
│  MCP Client  │ ◄──────────────►  │ MCP Bridge  │ ──────────────────► │  Target IDE  │
│ (Gemini CLI) │                    │  (Go server) │                    │ (Antigravity)│
└──────────────┘                    └──────┬──────┘                    └──────────────┘
                                           │
                                    ┌──────▼──────┐
                                    │  git status  │
                                    │  (polling)   │
                                    └─────────────┘
```

1. **Clipboard** — Copies prompt via `pbcopy`
2. **Activate** — Brings target app to foreground via AppleScript
3. **Chat** — Opens chat panel (`Cmd+shortcut`), clears, pastes, presses Enter
4. **Poll** — Watches `git status --porcelain` for workspace changes
5. **Report** — Returns diff, file status, and new file previews

---

## MCP Tools

### `send_to_app`

Sends a prompt to the target application and waits for workspace changes.

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `prompt` | ✅ | — | The prompt/task to send |
| `workspacePath` | — | `MCP_BRIDGE_WORKSPACE` | Workspace directory to monitor |
| `waitSeconds` | — | `120` | Max seconds to wait for changes |

**Returns:** Markdown-formatted result with prompt echo, change detection status, and full workspace diff.

### `check_workspace_changes`

Checks current workspace state without sending a prompt.

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `workspacePath` | — | `MCP_BRIDGE_WORKSPACE` | Workspace directory to check |

**Returns:** Git status, diff (truncated at 5KB), untracked file previews (truncated at 500 chars each), and 3 most recent commits.

---

## Quick Start

### Build

```sh
git clone https://github.com/jangtrinh/mcp-bridge-go.git
cd mcp-bridge-go
go build -o mcp-bridge .
```

### Configure

All configuration is via environment variables — no config files needed:

| Variable | Default | Description |
|----------|---------|-------------|
| `MCP_BRIDGE_APP` | `Antigravity` | Target application name |
| `MCP_BRIDGE_SHORTCUT` | `l` | Key for chat panel (used with Cmd) |
| `MCP_BRIDGE_WORKSPACE` | `.` (current dir) | Default workspace path |

### Supported Apps

| App | `MCP_BRIDGE_APP` | `MCP_BRIDGE_SHORTCUT` |
|-----|-------------------|-----------------------|
| Antigravity | `Antigravity` | `l` |
| Cursor | `Cursor` | `l` |
| Windsurf | `Windsurf` | `l` |

### Add to Gemini CLI

`~/.gemini/settings.json`:

```json
{
  "mcpServers": {
    "mcp-bridge": {
      "command": "/path/to/mcp-bridge",
      "env": {
        "MCP_BRIDGE_APP": "Antigravity",
        "MCP_BRIDGE_WORKSPACE": "/path/to/your/project"
      }
    }
  }
}
```

### Add to Claude Desktop

`~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "mcp-bridge": {
      "command": "/path/to/mcp-bridge",
      "env": {
        "MCP_BRIDGE_APP": "Cursor",
        "MCP_BRIDGE_WORKSPACE": "/path/to/your/project"
      }
    }
  }
}
```

---

## Requirements

- **macOS** — uses AppleScript for UI automation
- **Go 1.23+** — to build from source
- **Git** — for workspace change detection
- Target app must be **open** with a workspace loaded
- **Accessibility permissions** — System Settings → Privacy & Security → Accessibility → enable your terminal/MCP client

---

## Limitations

- **macOS only** — AppleScript is not available on other platforms
- **Requires UI focus** — the target app is brought to the foreground during prompt submission
- **Git-based detection** — only detects changes in git-tracked workspaces
- **No response content** — captures file changes, not the AI's text response
- **Sequential only** — one prompt at a time (the app needs focus)

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines and code standards.

## License

MIT © 2026 Jang Trinh
