<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26-00ADD8?style=flat-square&logo=go&logoColor=white" />
  <img src="https://img.shields.io/badge/MCP-stdio-blue?style=flat-square" />
  <img src="https://img.shields.io/badge/macOS-AppleScript-black?style=flat-square&logo=apple&logoColor=white" />
  <img src="https://img.shields.io/badge/license-MIT-green?style=flat-square" />
</p>

<h1 align="center">🌉 MCP Bridge</h1>

<p align="center">
  <strong>Lightweight MCP server that bridges AI coding tools via macOS automation.</strong>
  <br/>
  <sub>Send prompts to any Electron-based AI IDE and get back git diffs — all through the MCP protocol.</sub>
</p>

<p align="center">
  <a href="#-the-problem">Problem</a> ·
  <a href="#-the-hack-paperclip--gemini-cli">The Hack</a> ·
  <a href="#-how-it-works">How It Works</a> ·
  <a href="#-quick-start">Quick Start</a> ·
  <a href="#-mcp-tools">API</a> ·
  <a href="#%EF%B8%8F-known-limitations">Limitations</a>
</p>

---

## 💡 The Problem

Modern AI-assisted development involves **multiple AI tools** — but they can't talk to each other. You use Gemini CLI for quick tasks, Antigravity for deep coding, Claude Desktop for review — and **you're the manual copy-paste bridge** between them.

We hit this trying to adopt [Paperclip](https://github.com/paperclipai/paperclip) for AI agent orchestration:

- **Paperclip doesn't support Antigravity.** It ships adapters for Claude Code, Codex, and Cursor — but not Antigravity.
- **API costs kill you.** Paperclip's default adapter uses Claude Code, which burns paid tokens on trivial routing tasks. Most orchestration doesn't need a frontier model.

---

## 🔧 The Hack: Paperclip → Gemini CLI

This is the fun part. Paperclip's agent adapter is **hardcoded for Claude Code** — it calls the agent binary with Claude-specific flags:

```
claude --print --verbose --max-turns 10 --add-dir /path/to/workspace "Do the task"
```

Gemini CLI doesn't understand any of these flags. It would just crash.

**The hack:** a 25-line shell script that pretends to be Claude.

```bash
#!/bin/bash
# Paperclip thinks it's calling Claude Code.
# It's not. It's calling Gemini CLI.

# Ignore ALL Claude-specific args ($@). Read prompt from stdin.
PROMPT=$(cat)

# Forward to Gemini CLI with Gemini-native flags
exec gemini --prompt "$PROMPT" --yolo --output-format stream-json
```

That's it. Paperclip sends a heartbeat with the prompt on stdin + Claude flags as positional args. The wrapper **ignores** the args and **pipes** stdin straight to Gemini CLI. Paperclip has no idea it's talking to a different model.

### Why this matters

```
┌─────────────┐    Paperclip     ┌─────────────┐     MCP Bridge     ┌──────────────┐
│  Paperclip  │ ──────────────► │ Gemini CLI  │ ──────────────────► │ Antigravity  │
│  (CEO agent)│   heartbeat      │ (free tier) │    AppleScript      │ (heavy code) │
└─────────────┘                  └─────────────┘                    └──────────────┘
                                  🆓 Free tokens                     🧠 Deep coding
                                  ⚡ Fast routing                    📝 File changes
                                  🔧 Coordination                    🔍 Git diffs
```

| Layer | What | Cost |
|-------|------|------|
| **Paperclip** | Task management, agent governance | Free (self-hosted) |
| **gemini-wrapper.sh** | The 25-line hack — translates Claude → Gemini | Free |
| **Gemini CLI** | AI orchestration via Gemini 2.5 Pro | Free (1,000 req/day) |
| **MCP Bridge** | macOS automation + git polling | Free (local binary) |
| **Antigravity** | The actual coding | Your subscription |

**Total orchestration cost: $0/day for up to 1,000 tasks.**

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
| **4. Poll** | Watches `git status --porcelain` until changes appear |
| **5. Stability check** | After detecting changes, polls 3 more times to confirm IDE stopped writing |
| **6. Report** | Returns diff, file status, and new file previews |

### Stability Check

A naive approach would read the diff as soon as `git status` changes — but the IDE might still be writing files. MCP Bridge uses an **active stability check**: after detecting changes, it polls `git status` 3 consecutive times (every 2s). Only when the status is identical across all 3 checks does it consider the IDE done. If the status keeps changing, the counter resets.

```
Change detected → poll #1 (2s) → same? ✓ → poll #2 (2s) → same? ✓ → poll #3 (2s) → same? ✓ → STABLE → read diff
                                  └─ different? → reset counter, keep polling
```

---

## 🚀 Quick Start

### Build

```sh
git clone https://github.com/jangtrinh/mcp-bridge-go.git
cd mcp-bridge-go
make build
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

### Use with Paperclip

**Step 1:** Build MCP Bridge and add it to Gemini CLI (see above).

**Step 2:** In Paperclip, set the agent command to the adapter:

```
/path/to/mcp-bridge-go/scripts/paperclip-adapter.sh
```

**Step 3:** Start Paperclip, open Antigravity with your project, assign a task. The full chain activates on the next heartbeat:

```
Paperclip CEO → paperclip-adapter.sh → Gemini CLI (free) → MCP Bridge → Antigravity
     │                   │                     │                   │
     │                   │                     │                   └── Writes code, edits files
     │                   │                     └── Has MCP Bridge as MCP server
     │                   └── Ignores Claude flags, pipes stdin to Gemini
     └── Sends heartbeat with prompt on stdin
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

**On success:** Returns markdown with prompt echo, change status, and full workspace diff.

**On timeout:** Returns diagnostic report — app state (running/frontmost), workspace state, and recommended next actions.

### `check_workspace_changes`

Checks current workspace state without sending a prompt.

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `workspacePath` | — | `MCP_BRIDGE_WORKSPACE` | Workspace to check |

**Returns:** Git status, diff (≤5KB), untracked file previews (≤500 chars each), and 3 most recent commits.

---

## ⚠️ Known Limitations

### Git Polling as Completion Signal

MCP Bridge uses `git status` to detect when the target AI has finished. This is the biggest architectural trade-off:

| Scenario | Git status | Bridge behavior |
|----------|-----------|----------------|
| AI writes code | Changes | ✅ Stability check → returns diff |
| AI asks a clarifying question | No changes | ⏱ Timeout with diagnostic |
| AI says "no changes needed" | No changes | ⏱ Timeout with diagnostic |
| AI crashes | No changes | ⏱ Timeout with diagnostic |

**Why?** Antigravity (and similar IDEs) don't expose an API to read AI responses. The GUI is the only interface. Git polling detects *side effects* (file changes) but is blind to *text responses*.

#### Timeout Diagnostics

When no changes are detected, MCP Bridge returns an informative diagnostic instead of failing silently:

```
⏱ Prompt sent to Antigravity but no file changes detected after 120s.

Diagnostic:
- Antigravity: running and frontmost (may still be processing)
- Workspace: clean (no uncommitted changes)
- Wait time: 120s

Recommended actions:
- The AI may have responded with a question or explanation (no code changes)
- Try sending a more specific prompt
- Use `check_workspace_changes` to verify current state
- Increase `waitSeconds` if the task is complex
```

### Other Limitations

- **macOS only** — AppleScript is not available on other platforms
- **Requires UI focus** — target app comes to foreground during submission
- **Sequential only** — one prompt at a time (needs app focus)
- **No response content** — captures file changes, not the AI's text response

---

## 🔄 Alternatives

| Tool | Approach | Trade-off |
|------|----------|-----------|
| **SimulateDev** | Full workflow engine, AppleScript-based | Heavier — full Planner/Coder/Tester pipeline |
| **Claude Agent Teams** | Claude agents via `@mentions` | Locked to Claude ecosystem |
| **applescript-mcp** | Generic AppleScript MCP server | General-purpose, no AI-IDE-specific features |
| **VS Code Subagents** | VS Code's native delegation | VS Code only, not cross-tool |
| **MCP Bridge** | MCP-native, ~560 lines, cross-tool | macOS only, sequential, requires UI focus |

**MCP Bridge is for you if:** You want a tiny, composable MCP tool that connects any MCP client to any Electron-based AI IDE — especially if you're optimizing for free-tier models as orchestrators.

---

## 🛠️ Development

```sh
make audit    # Runs: gofmt → go vet → golangci-lint → gosec → go build
make build    # Compile with version/commit/date via ldflags
make lint     # Static analysis only
make clean    # Remove binary
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for code standards and PR guidelines.

## 📋 Requirements

- **macOS** — uses AppleScript for UI automation
- **Go 1.26+** — to build from source
- **Git** — for workspace change detection
- Target app must be **open** with a workspace loaded
- **Accessibility permissions** — System Settings → Privacy & Security → Accessibility → enable your terminal/MCP client

## 📄 License

MIT © 2026 Jang Trinh
