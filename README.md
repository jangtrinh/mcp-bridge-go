<p align="center">
  <img src="https://img.shields.io/badge/Go-1.23-00ADD8?style=flat-square&logo=go&logoColor=white" />
  <img src="https://img.shields.io/badge/MCP-stdio-blue?style=flat-square" />
  <img src="https://img.shields.io/badge/macOS-AppleScript-black?style=flat-square&logo=apple&logoColor=white" />
  <img src="https://img.shields.io/badge/license-MIT-green?style=flat-square" />
</p>

<h1 align="center">🌉 MCP Bridge</h1>

<p align="center">
  <strong>Lightweight MCP server that bridges AI coding tools via macOS automation.</strong>
  <br/>
  <sub>Send prompts to any Electron-based AI IDE and monitor workspace changes — all through the MCP protocol.</sub>
</p>

<p align="center">
  <a href="#-motivation">Motivation</a> ·
  <a href="#-use-cases">Use Cases</a> ·
  <a href="#-quick-start">Quick Start</a> ·
  <a href="#-mcp-tools">API</a> ·
  <a href="#-alternatives">Alternatives</a> ·
  <a href="CONTRIBUTING.md">Contributing</a>
</p>

---

## 💡 Motivation

Modern AI-assisted development involves **multiple AI tools** — but they can't talk to each other. You use Gemini CLI for quick tasks, Antigravity for deep coding, Claude Desktop for review — and **you're the manual copy-paste bridge**.

We hit this exact problem trying to adopt [Paperclip](https://github.com/paperclipai/paperclip) for AI agent orchestration:

**Problem 1: Paperclip doesn't support Antigravity.** It has adapters for Claude Code, Codex, and Cursor — but not Antigravity. We needed a bridge.

**Problem 2: API costs.** Paperclip's default adapter uses Claude Code, which burns through paid tokens fast on simple routing and coordination tasks. Most orchestration doesn't need a frontier model — it needs a reliable, fast model with a generous free tier.

**The solution:**

```
┌─────────────┐    Paperclip     ┌─────────────┐     MCP Bridge     ┌──────────────┐
│  Paperclip  │ ──────────────► │ Gemini CLI  │ ──────────────────► │ Antigravity  │
│  (CEO agent)│   heartbeat      │ (free tier) │    AppleScript      │ (heavy code) │
└─────────────┘                  └─────────────┘                    └──────────────┘
                                  🆓 Free tokens                     🧠 Deep coding
                                  ⚡ Fast routing                    📝 File changes
                                  🔧 Coordination                    🔍 Git diffs
```

**Gemini CLI** (free tier — 1,000 req/day, 2.5 Pro) handles orchestration and routing. **MCP Bridge** delegates heavy coding to Antigravity (or Cursor/Windsurf). **Zero API cost for the orchestration layer.**

---

## 🎯 Use Cases

### 1. Paperclip → Gemini CLI → Antigravity (Our Setup)

The workflow that inspired this project — using Paperclip for agent companies with Gemini as the free orchestrator:

```
Paperclip CEO Agent
  └─► Assigns task to CTO agent (Gemini CLI, free tier)
       └─► CTO uses MCP Bridge to delegate to Antigravity
            └─► Antigravity writes the code
                 └─► CTO reviews via check_workspace_changes
                      └─► Reports completion back to CEO
```

**Why this works:** Paperclip doesn't support Antigravity natively. Gemini CLI is free. MCP Bridge fills the gap.

### 2. Gemini CLI → Any AI IDE

Direct delegation without Paperclip — use Gemini CLI as your orchestrator:

```
You (to Gemini CLI): "Refactor the auth module to use JWT tokens"
  └─► Gemini calls send_to_app with the prompt
       └─► MCP Bridge pastes the prompt into Antigravity
            └─► Antigravity processes the task, edits files
                 └─► MCP Bridge detects git changes, returns diff
                      └─► Gemini reviews and reports back
```

### 3. Cross-IDE Code Review

Use one AI tool to review another's work:

```
Claude Desktop
  └─► Sends coding task to Cursor via MCP Bridge
       └─► Cursor generates the code
            └─► Claude reads the diff and provides feedback
```

### 4. Automated Task Pipelines

Chain delegations — planning in one tool, implementation in another:

```
Orchestrator
  ├── Step 1: "Plan the database schema" → reviews diff
  ├── Step 2: "Implement the schema" → verifies output
  └── Step 3: "Write tests for the DB layer" → final check
```

---

## ⚙️ How It Works

```
┌──────────────┐     MCP stdio     ┌─────────────┐     AppleScript     ┌──────────────┐
│  MCP Client  │ ◄───────────────► │ MCP Bridge  │ ──────────────────► │  Target IDE  │
│ (Gemini CLI) │                   │  (Go server) │                    │ (Antigravity)│
└──────────────┘                   └──────┬──────┘                    └──────────────┘
                                          │
                                   ┌──────▼──────┐
                                   │  git status  │
                                   │  (polling)   │
                                   └─────────────┘
```

| Step | What happens |
|------|-------------|
| **1. Clipboard** | Copies prompt via `pbcopy` |
| **2. Activate** | Brings target app to foreground via AppleScript |
| **3. Chat** | Opens chat panel (`Cmd+shortcut`), clears, pastes, Enter |
| **4. Poll** | Watches `git status --porcelain` for changes |
| **5. Report** | Returns diff, file status, and new file previews |

---

## 🚀 Quick Start

### Build

```sh
git clone https://github.com/jangtrinh/mcp-bridge-go.git
cd mcp-bridge-go
go build -o mcp-bridge .
```

### Configure

All configuration via environment variables — zero config files:

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

> **💡 Works with any Electron-based AI IDE** that has a keyboard shortcut to open a chat panel.

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

## 🔧 MCP Tools

### `send_to_app`

Sends a prompt to the target application and waits for workspace changes.

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `prompt` | ✅ | — | The prompt/task to send |
| `workspacePath` | — | `MCP_BRIDGE_WORKSPACE` | Workspace to monitor |
| `waitSeconds` | — | `120` | Max wait time (seconds) |

**Returns:** Markdown result with prompt echo, change detection status, and full workspace diff.

### `check_workspace_changes`

Checks current workspace state without sending a prompt.

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `workspacePath` | — | `MCP_BRIDGE_WORKSPACE` | Workspace to check |

**Returns:** Git status, diff (≤5KB), untracked file previews (≤500 chars each), and 3 most recent commits.

---

## 🔄 Alternatives

We built MCP Bridge because nothing else fit our exact needs. Here's how it compares:

| Tool | Approach | Trade-off |
|------|----------|-----------|
| **SimulateDev** | Full workflow engine, AppleScript-based | Heavier — full Planner/Coder/Tester pipeline, not a simple MCP tool |
| **Claude Code Agent Teams** | Claude agents coordinate via `@mentions` | Locked to Claude ecosystem, doesn't bridge across tools |
| **applescript-mcp** | Generic AppleScript MCP server | General-purpose (Finder, Mail, etc.) — not AI-IDE-specific, no git monitoring |
| **VS Code Subagents** | VS Code's native delegation feature | VS Code ecosystem only, not cross-tool |
| **MCP Bridge** | MCP-native, 300 lines, cross-tool | macOS only, requires UI focus, sequential |

**MCP Bridge is for you if:** You want a tiny, composable MCP tool that connects any MCP client to any Electron-based AI IDE — especially if you're optimizing for free-tier models as orchestrators.

---

## 📋 Requirements

- **macOS** — uses AppleScript for UI automation
- **Go 1.23+** — to build from source
- **Git** — for workspace change detection
- Target app must be **open** with a workspace loaded
- **Accessibility permissions** — System Settings → Privacy & Security → Accessibility → enable your terminal/MCP client

## ⚠️ Limitations

- **macOS only** — AppleScript is not available on other platforms
- **Requires UI focus** — target app comes to foreground during submission
- **Git-based detection** — only works in git-tracked workspaces
- **No response content** — captures file changes, not the AI's text response
- **Sequential only** — one prompt at a time (needs app focus)

---

## 🤝 Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines, code standards, and quality gates.

## 📄 License

MIT © 2026 Jang Trinh
