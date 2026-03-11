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
  <a href="#-use-cases">Use Cases</a> ·
  <a href="#-quick-start">Quick Start</a> ·
  <a href="#-mcp-tools">API</a> ·
  <a href="CONTRIBUTING.md">Contributing</a>
</p>

---

## 💡 Motivation

We use [Paperclip](https://github.com/paperclipai/paperclip) to orchestrate AI agent teams — but Paperclip's adapters default to **Claude Code**, which burns through paid API tokens fast on routine tasks like delegation, status checks, and simple coordination.

**The insight:** Most orchestration tasks don't need a frontier model. They need a reliable, fast model with a generous free tier.

**The solution:** Use **Gemini CLI** (free tier, 2.5 Pro) as the orchestrator brain, and **MCP Bridge** to delegate heavy coding tasks to specialized AI IDEs (Antigravity, Cursor, etc.) that have their own model access.

```
┌─────────────┐    Paperclip     ┌─────────────┐     MCP Bridge     ┌──────────────┐
│  Paperclip  │ ──────────────► │ Gemini CLI  │ ──────────────────► │ Antigravity  │
│  (CEO agent)│   heartbeat      │ (free tier) │    AppleScript      │ (heavy code) │
└─────────────┘                  └─────────────┘                    └──────────────┘
                                  🆓 Free tokens                     🧠 Deep coding
                                  ⚡ Fast routing                    📝 File changes
                                  🔧 Task mgmt                      🔍 Git diffs
```

**Result:** Paperclip orchestrates. Gemini routes for free. Antigravity does the heavy lifting. **Zero API cost for orchestration.**

---

## 🎯 Use Cases

### 1. Gemini CLI → Antigravity Delegation

You're using Gemini CLI for orchestration but want Antigravity to handle complex code generation:

```
You (to Gemini CLI): "Refactor the auth module to use JWT tokens"
  └─► Gemini CLI calls send_to_app with the prompt
       └─► MCP Bridge pastes the prompt into Antigravity's chat
            └─► Antigravity processes the task, edits files
                 └─► MCP Bridge detects git changes, returns diff to Gemini
                      └─► Gemini reviews the changes and reports back
```

### 2. Multi-Agent Company with Paperclip

Use Paperclip to run AI agent companies where a CEO agent delegates coding tasks:

```
Paperclip CEO Agent
  └─► Assigns task to CTO agent (Gemini CLI, free tier)
       └─► CTO uses MCP Bridge to delegate to Antigravity
            └─► Antigravity writes the code
                 └─► CTO reviews via check_workspace_changes
                      └─► Reports completion back to CEO
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
Orchestrator AI
  ├── Step 1: send_to_app → "Plan the database schema"
  │    └─► Reviews the plan from the diff
  ├── Step 2: send_to_app → "Implement the schema"
  │    └─► Verifies implementation matches plan
  └── Step 3: send_to_app → "Write tests for the DB layer"
       └─► Final verification
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
