// Package main implements an MCP (Model Context Protocol) server that bridges
// AI coding tools via macOS AppleScript automation. It enables any MCP client
// to delegate tasks to Electron-based AI IDEs (Antigravity, Cursor, Windsurf)
// and monitor workspace changes through git.
//
// Build with version info:
//
//	go build -ldflags "-X main.version=1.2.0 -X main.commit=$(git rev-parse --short HEAD) -X main.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Build-time variables, injected via -ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// ─── CONFIGURATION ──────────────────────────────────────────────

// Config holds runtime configuration, loaded from environment variables.
// All fields have sensible defaults for Antigravity on macOS.
type Config struct {
	AppName     string // Target application name (MCP_BRIDGE_APP, default: "Antigravity")
	ShortcutKey string // Key to open chat panel with Cmd (MCP_BRIDGE_SHORTCUT, default: "l")
	Workspace   string // Default workspace path (MCP_BRIDGE_WORKSPACE, default: cwd)
	WaitSeconds int    // Default seconds to wait for changes (default: 120)
}

// loadConfig reads configuration from environment variables with sensible defaults.
func loadConfig() Config {
	cfg := Config{
		AppName:     envOr("MCP_BRIDGE_APP", "Antigravity"),
		ShortcutKey: envOr("MCP_BRIDGE_SHORTCUT", "l"),
		Workspace:   envOr("MCP_BRIDGE_WORKSPACE", "."),
		WaitSeconds: 120,
	}

	// Resolve workspace to absolute path for predictable behavior.
	if !filepath.IsAbs(cfg.Workspace) {
		if abs, err := filepath.Abs(cfg.Workspace); err == nil {
			cfg.Workspace = abs
		}
	}

	return cfg
}

// validate checks configuration invariants and returns an error if invalid.
func (c Config) validate() error {
	if c.AppName == "" {
		return fmt.Errorf("MCP_BRIDGE_APP must not be empty")
	}
	if c.ShortcutKey == "" {
		return fmt.Errorf("MCP_BRIDGE_SHORTCUT must not be empty")
	}
	if c.WaitSeconds < 5 {
		return fmt.Errorf("wait seconds must be at least 5, got %d", c.WaitSeconds)
	}
	if _, err := os.Stat(c.Workspace); err != nil {
		return fmt.Errorf("workspace %q: %w", c.Workspace, err)
	}
	return nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ─── WAIT RESULT ────────────────────────────────────────────────

// waitResult captures the outcome of polling for workspace changes,
// including diagnostic context when no changes are detected (timeout).
type waitResult struct {
	Detected   bool   // Whether file changes were detected
	Reason     string // Human-readable explanation of why wait ended
	Diagnostic string // Contextual info for the caller on timeout
}

// ─── APPLESCRIPT HELPERS ────────────────────────────────────────

const (
	maxRetries    = 3
	retryInterval = 500 * time.Millisecond
)

// runOsascript executes a single-line AppleScript and returns its output.
func runOsascript(script string) (string, error) {
	cmd := exec.Command("osascript", "-e", script) // #nosec G204 — AppleScript is core functionality
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// runOsascriptMulti executes a multi-line AppleScript (each element is one -e flag).
func runOsascriptMulti(lines []string) error {
	args := make([]string, 0, len(lines)*2)
	for _, l := range lines {
		args = append(args, "-e", l)
	}
	cmd := exec.Command("osascript", args...) // #nosec G204 — AppleScript is core functionality
	_, err := cmd.CombinedOutput()
	return err
}

// runOsascriptWithRetry wraps runOsascriptMulti with exponential backoff.
// AppleScript calls can fail transiently when the target app is busy.
func runOsascriptWithRetry(lines []string) error {
	var lastErr error
	for attempt := range maxRetries {
		if err := runOsascriptMulti(lines); err != nil {
			lastErr = err
			slog.Warn("AppleScript attempt failed, retrying",
				"attempt", attempt+1,
				"max", maxRetries,
				"error", err,
			)
			time.Sleep(retryInterval * time.Duration(attempt+1))
			continue
		}
		return nil
	}
	return fmt.Errorf("AppleScript failed after %d attempts: %w", maxRetries, lastErr)
}

// isAppRunning checks whether the target application process exists.
func isAppRunning(appName string) bool {
	out, err := runOsascript(`application "` + appName + `" is running`)
	return err == nil && out == "true"
}

// isAppFrontmost checks whether the target application is the frontmost window.
func isAppFrontmost(appName string) bool {
	out, err := runOsascript(`tell application "System Events" to get name of first application process whose frontmost is true`)
	return err == nil && strings.EqualFold(strings.TrimSpace(out), appName)
}

// sendPrompt automates the target app's UI to submit a prompt:
// clipboard → activate → open chat → clear → paste → enter.
func sendPrompt(cfg Config, prompt string) error {
	// Set clipboard via pbcopy.
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(prompt)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pbcopy: %w", err)
	}

	// Activate app.
	if _, err := runOsascript(`tell application "` + cfg.AppName + `" to activate`); err != nil {
		return fmt.Errorf("activate %s: %w", cfg.AppName, err)
	}
	time.Sleep(500 * time.Millisecond)

	// Open chat panel (Cmd+shortcutKey).
	if err := runOsascriptWithRetry([]string{
		`tell application "System Events"`,
		`  keystroke "` + cfg.ShortcutKey + `" using {command down}`,
		`end tell`,
	}); err != nil {
		return fmt.Errorf("open chat: %w", err)
	}
	time.Sleep(1 * time.Second)

	// Select all + delete (clear existing input).
	if err := runOsascriptWithRetry([]string{
		`tell application "System Events"`,
		`  keystroke "a" using {command down}`,
		`  delay 0.2`,
		`  key code 51`,
		`end tell`,
	}); err != nil {
		return fmt.Errorf("clear input: %w", err)
	}
	time.Sleep(300 * time.Millisecond)

	// Paste prompt (Cmd+V).
	if err := runOsascriptWithRetry([]string{
		`tell application "System Events"`,
		`  keystroke "v" using {command down}`,
		`end tell`,
	}); err != nil {
		return fmt.Errorf("paste: %w", err)
	}
	time.Sleep(300 * time.Millisecond)

	// Press Enter to submit.
	if err := runOsascriptWithRetry([]string{
		`tell application "System Events"`,
		`  key code 36`,
		`end tell`,
	}); err != nil {
		return fmt.Errorf("enter: %w", err)
	}

	return nil
}

// ─── GIT HELPERS ────────────────────────────────────────────────

// gitCmd runs a git command in the given workspace and returns stdout.
// Returns an error if the command fails (git not installed, not a repo, etc.).
func gitCmd(workspace string, args ...string) (string, error) {
	cmd := exec.Command("git", args...) // #nosec G204 — git invoked with controlled args
	cmd.Dir = workspace
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w (output: %s)", args[0], err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// gitCmdSafe runs a git command and returns the output, logging and returning
// empty string on error. Use when errors are non-critical (reporting, display).
func gitCmdSafe(workspace string, args ...string) string {
	out, err := gitCmd(workspace, args...)
	if err != nil {
		slog.Warn("git command failed", "args", args, "error", err)
		return ""
	}
	return out
}

// gitStatus returns the porcelain status of the workspace (one line per changed file).
// Returns an error if git fails, allowing callers to distinguish "clean" from "broken".
func gitStatus(workspace string) (string, error) {
	return gitCmd(workspace, "status", "--porcelain")
}

// getWorkspaceChanges compiles a comprehensive workspace report:
// file status, diff (capped at 5KB), untracked file previews, and recent commits.
// Uses git repo root for file paths to handle subdirectory workspaces correctly.
func getWorkspaceChanges(workspace string) string {
	var parts []string

	// Resolve repo root — git returns paths relative to this, not to workspace.
	repoRoot := gitCmdSafe(workspace, "rev-parse", "--show-toplevel")
	if repoRoot == "" {
		repoRoot = workspace // Fallback if rev-parse fails.
	}

	if status := gitCmdSafe(workspace, "status", "--short"); status != "" {
		parts = append(parts, "=== File Status ===\n"+status)
	}

	if diff := gitCmdSafe(workspace, "diff"); diff != "" {
		const maxDiffBytes = 5000
		if len(diff) > maxDiffBytes {
			diff = diff[:maxDiffBytes] + "\n... (truncated)"
		}
		parts = append(parts, "=== Changes (diff) ===\n"+diff)
	}

	if untracked := gitCmdSafe(workspace, "ls-files", "--others", "--exclude-standard"); untracked != "" {
		files := strings.Split(untracked, "\n")
		var previews []string
		for _, f := range files {
			if f == "" {
				continue
			}
			fullPath := filepath.Join(repoRoot, filepath.Clean(f))
			content, err := os.ReadFile(fullPath) // #nosec G304 — workspace is user-configured
			if err != nil {
				previews = append(previews, fmt.Sprintf("--- %s --- (unreadable)", f))
				continue
			}
			const maxPreviewBytes = 500
			preview := string(content)
			if len(preview) > maxPreviewBytes {
				preview = preview[:maxPreviewBytes] + "\n... (truncated)"
			}
			previews = append(previews, fmt.Sprintf("--- %s ---\n%s", f, preview))
		}
		parts = append(parts, "=== New Files ===\n"+strings.Join(previews, "\n\n"))
	}

	if recentLog := gitCmdSafe(workspace, "log", "--oneline", "-3", "--format=%h %s (%cr)"); recentLog != "" {
		parts = append(parts, "=== Recent Commits ===\n"+recentLog)
	}

	return strings.Join(parts, "\n\n")
}

// waitForChanges polls git status until either the workspace changes or the
// deadline expires. After detecting a change, it performs a stability check:
// polling git status multiple consecutive times to ensure the IDE has fully
// stopped writing before reading the diff.
func waitForChanges(ctx context.Context, cfg Config, workspace, initialStatus string, waitSecs int) waitResult {
	const (
		initialDelay        = 5 * time.Second
		pollInterval        = 3 * time.Second
		settleInterval      = 2 * time.Second
		stableChecksRequired = 3
	)

	// Wait for app to start processing.
	select {
	case <-ctx.Done():
		return waitResult{Reason: "cancelled"}
	case <-time.After(initialDelay):
	}

	deadline := time.Now().Add(time.Duration(waitSecs) * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return waitResult{Reason: "cancelled"}
		default:
		}

		current, err := gitStatus(workspace)
		if err != nil {
			slog.Warn("git status failed during poll, retrying", "error", err)
			time.Sleep(pollInterval)
			continue
		}
		if current != initialStatus {
			// Change detected — run stability check.
			// Keep polling until git status is unchanged for N consecutive checks.
			slog.Info("change detected, starting stability check")
			stableCount := 0
			lastStatus := current

			for stableCount < stableChecksRequired && time.Now().Before(deadline) {
				select {
				case <-ctx.Done():
					return waitResult{Reason: "cancelled"}
				default:
				}

				time.Sleep(settleInterval)
				newStatus, err := gitStatus(workspace)
				if err != nil {
					slog.Warn("git status failed during stability check", "error", err)
					continue
				}

				if newStatus == lastStatus {
					stableCount++
					slog.Debug("stability check pass",
						"stable_count", stableCount,
						"required", stableChecksRequired,
					)
				} else {
					// Still changing — reset counter.
					slog.Info("workspace still changing, resetting stability counter")
					stableCount = 0
					lastStatus = newStatus
				}
			}

			return waitResult{
				Detected: true,
				Reason:   "File changes detected (stable)",
			}
		}
		time.Sleep(pollInterval)
	}

	// Timeout — gather diagnostic info.
	appState := "not running"
	if isAppRunning(cfg.AppName) {
		if isAppFrontmost(cfg.AppName) {
			appState = "running and frontmost (may still be processing or waiting for input)"
		} else {
			appState = "running but not frontmost (another app took focus)"
		}
	}

	workspaceState := "clean (no uncommitted changes)"
	if s, err := gitStatus(workspace); err == nil && s != "" {
		workspaceState = "has uncommitted changes (same as before prompt)"
	}

	diagnostic := fmt.Sprintf("- %s: %s\n- Workspace: %s\n- Wait time: %ds",
		cfg.AppName, appState, workspaceState, waitSecs)

	return waitResult{
		Detected:   false,
		Reason:     fmt.Sprintf("No file changes after %ds", waitSecs),
		Diagnostic: diagnostic,
	}
}

// ─── TOOL HANDLERS ──────────────────────────────────────────────

// handleSendToApp sends a prompt to the target AI IDE, waits for workspace
// changes, and returns either the diff or a diagnostic timeout report.
func handleSendToApp(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	prompt, err := req.RequireString("prompt")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter: prompt"), nil
	}

	workspace := req.GetString("workspacePath", cfg.Workspace)
	waitSecs := int(req.GetFloat("waitSeconds", float64(cfg.WaitSeconds)))

	slog.Info("send_to_app called",
		"app", cfg.AppName,
		"workspace", workspace,
		"prompt_len", len(prompt),
		"wait_seconds", waitSecs,
	)

	// Step 1: Preflight — is the app running?
	if !isAppRunning(cfg.AppName) {
		return mcp.NewToolResultError(fmt.Sprintf("%s is not running. Please open it first.", cfg.AppName)), nil
	}

	// Step 2: Snapshot workspace before sending.
	initialStatus, err := gitStatus(workspace)
	if err != nil {
		slog.Warn("git status failed for initial snapshot, using empty baseline", "error", err)
		initialStatus = ""
	}

	// Step 3: Send prompt via AppleScript UI automation.
	if err := sendPrompt(cfg, prompt); err != nil {
		slog.Error("send prompt failed", "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to send prompt: %v", err)), nil
	}
	slog.Info("prompt sent, waiting for changes")

	// Step 4: Wait for changes with diagnostic.
	wr := waitForChanges(ctx, cfg, workspace, initialStatus, waitSecs)

	// Step 5: Build result.
	if wr.Detected {
		changes := getWorkspaceChanges(workspace)
		slog.Info("changes detected", "reason", wr.Reason)
		result := fmt.Sprintf("✅ Prompt sent to %s\n\n"+
			"**Prompt:** %s\n\n"+
			"**Status:** %s\n\n"+
			"**Workspace changes:**\n```\n%s\n```",
			cfg.AppName, prompt, wr.Reason, changes)
		return mcp.NewToolResultText(result), nil
	}

	// Timeout — informative diagnostic instead of silent failure.
	slog.Warn("timeout waiting for changes",
		"reason", wr.Reason,
		"wait_seconds", waitSecs,
	)
	result := fmt.Sprintf("⏱ Prompt sent to %s but no file changes detected after %ds.\n\n"+
		"**Prompt:** %s\n\n"+
		"**Reason:** %s\n\n"+
		"**Diagnostic:**\n%s\n\n"+
		"**Recommended actions:**\n"+
		"- The AI may have responded with a question or explanation (no code changes)\n"+
		"- Try sending a more specific prompt\n"+
		"- Use `check_workspace_changes` to verify current state\n"+
		"- Increase `waitSeconds` if the task is complex",
		cfg.AppName, waitSecs, prompt, wr.Reason, wr.Diagnostic)
	return mcp.NewToolResultText(result), nil
}

// handleCheckChanges returns the current workspace state without sending a prompt.
func handleCheckChanges(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	workspace := req.GetString("workspacePath", cfg.Workspace)
	changes := getWorkspaceChanges(workspace)

	if changes == "" {
		return mcp.NewToolResultText("Workspace is clean. No changes."), nil
	}
	return mcp.NewToolResultText(changes), nil
}

// ─── MAIN ───────────────────────────────────────────────────────

var cfg Config

func main() {
	// Structured logging to stderr (stdout is reserved for MCP stdio).
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg = loadConfig()
	if err := cfg.validate(); err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	slog.Info("mcp-bridge starting",
		"version", version,
		"commit", commit,
		"date", date,
		"app", cfg.AppName,
		"workspace", cfg.Workspace,
	)

	s := server.NewMCPServer(
		"mcp-bridge",
		version,
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	// Tool 1: send_to_app — delegate a prompt to the target IDE.
	s.AddTool(
		mcp.NewTool("send_to_app",
			mcp.WithDescription(fmt.Sprintf(
				"Send a prompt/task to %s AI coding assistant. "+
					"Automates the app UI to submit your prompt, then monitors workspace for file changes. "+
					"%s must be open with a workspace loaded.", cfg.AppName, cfg.AppName)),
			mcp.WithString("prompt",
				mcp.Required(),
				mcp.Description("The prompt/task to send. Be specific and detailed."),
			),
			mcp.WithString("workspacePath",
				mcp.Description(fmt.Sprintf("Absolute path to workspace directory. Default: %s", cfg.Workspace)),
			),
			mcp.WithNumber("waitSeconds",
				mcp.Description("Max seconds to wait for completion. Default: 120"),
			),
		),
		handleSendToApp,
	)

	// Tool 2: check_workspace_changes — inspect workspace state.
	s.AddTool(
		mcp.NewTool("check_workspace_changes",
			mcp.WithDescription("Check what files changed in the workspace. Returns git status, diffs, and new file previews."),
			mcp.WithString("workspacePath",
				mcp.Description(fmt.Sprintf("Absolute path to workspace. Default: %s", cfg.Workspace)),
			),
		),
		handleCheckChanges,
	)

	// Graceful shutdown on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		slog.Info("shutdown signal received, exiting")
		os.Exit(0)
	}()

	// Start MCP stdio transport.
	slog.Info("serving MCP over stdio")
	if err := server.ServeStdio(s); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
