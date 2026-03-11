package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Config holds runtime configuration, loaded from environment variables.
type Config struct {
	AppName     string // Target application name (default: "Antigravity")
	ShortcutKey string // Key to open chat panel, used with Cmd (default: "l")
	Workspace   string // Default workspace path (default: current dir)
	WaitSeconds int    // Default seconds to wait for changes (default: 120)
}

func loadConfig() Config {
	cfg := Config{
		AppName:     envOr("MCP_BRIDGE_APP", "Antigravity"),
		ShortcutKey: envOr("MCP_BRIDGE_SHORTCUT", "l"),
		Workspace:   envOr("MCP_BRIDGE_WORKSPACE", "."),
		WaitSeconds: 120,
	}

	// Resolve workspace to absolute path
	if !filepath.IsAbs(cfg.Workspace) {
		if abs, err := filepath.Abs(cfg.Workspace); err == nil {
			cfg.Workspace = abs
		}
	}

	return cfg
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

var cfg Config

func main() {
	cfg = loadConfig()

	s := server.NewMCPServer(
		"mcp-bridge",
		"1.1.0",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	// Tool 1: send_to_app
	sendTool := mcp.NewTool("send_to_app",
		mcp.WithDescription(fmt.Sprintf(
			"Send a prompt/task to %s AI coding assistant. "+
				"Automates the app UI to submit your prompt, then monitors workspace for file changes. "+
				"%s must be open with a workspace loaded.", cfg.AppName, cfg.AppName)),
		mcp.WithString("prompt",
			mcp.Required(),
			mcp.Description("The prompt/task to send. Be specific and detailed."),
		),
		mcp.WithString("workspacePath",
			mcp.Description(fmt.Sprintf("Absolute path to the workspace directory. Default: %s", cfg.Workspace)),
		),
		mcp.WithNumber("waitSeconds",
			mcp.Description("Max seconds to wait for completion. Default: 120"),
		),
	)
	s.AddTool(sendTool, handleSendToApp)

	// Tool 2: check_workspace_changes
	checkTool := mcp.NewTool("check_workspace_changes",
		mcp.WithDescription("Check what files changed in the workspace. Returns git status, diffs, and new file previews."),
		mcp.WithString("workspacePath",
			mcp.Description(fmt.Sprintf("Absolute path to workspace. Default: %s", cfg.Workspace)),
		),
	)
	s.AddTool(checkTool, handleCheckChanges)

	// Start stdio server
	log.SetOutput(os.Stderr)
	log.Printf("🌉 MCP Bridge running — target: %s, workspace: %s\n", cfg.AppName, cfg.Workspace)
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// ─── WAIT RESULT ────────────────────────────────────────────────

type waitResult struct {
	Detected   bool   // Whether file changes were detected
	Reason     string // Why the wait ended
	Diagnostic string // Contextual info for the caller
}

// ─── TOOL HANDLERS ──────────────────────────────────────────────

func handleSendToApp(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	prompt, err := req.RequireString("prompt")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: prompt"), nil
	}

	workspace := req.GetString("workspacePath", cfg.Workspace)
	waitSecs := int(req.GetFloat("waitSeconds", float64(cfg.WaitSeconds)))

	// Step 1: Check app is running
	if !isAppRunning() {
		return mcp.NewToolResultError(fmt.Sprintf("%s is not running. Please open it first.", cfg.AppName)), nil
	}

	// Step 2: Snapshot workspace
	initialStatus := gitStatus(workspace)

	// Step 3: Send prompt via AppleScript
	if err := sendPrompt(prompt); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to send prompt: %v", err)), nil
	}

	// Step 4: Wait for changes with diagnostic
	wr := waitForChanges(workspace, initialStatus, waitSecs)

	// Step 5: Build result
	if wr.Detected {
		changes := getWorkspaceChanges(workspace)
		result := fmt.Sprintf("✅ Prompt sent to %s\n\n"+
			"**Prompt:** %s\n\n"+
			"**Status:** %s\n\n"+
			"**Workspace changes:**\n```\n%s\n```",
			cfg.AppName, prompt, wr.Reason, changes)
		return mcp.NewToolResultText(result), nil
	}

	// Timeout — return diagnostic instead of silent failure
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

func handleCheckChanges(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	workspace := req.GetString("workspacePath", cfg.Workspace)
	changes := getWorkspaceChanges(workspace)

	if changes == "" {
		return mcp.NewToolResultText("Workspace is clean. No changes."), nil
	}
	return mcp.NewToolResultText(changes), nil
}

// ─── APPLESCRIPT HELPERS ────────────────────────────────────────

func runOsascript(script string) (string, error) {
	cmd := exec.Command("osascript", "-e", script) // #nosec G204 — AppleScript execution is core functionality
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func runOsascriptMulti(lines []string) error {
	args := make([]string, 0, len(lines)*2)
	for _, l := range lines {
		args = append(args, "-e", l)
	}
	cmd := exec.Command("osascript", args...) // #nosec G204 — AppleScript execution is core functionality
	_, err := cmd.CombinedOutput()
	return err
}

func isAppRunning() bool {
	out, err := runOsascript(`application "` + cfg.AppName + `" is running`)
	return err == nil && out == "true"
}

func isAppFrontmost() bool {
	out, err := runOsascript(`tell application "System Events" to get name of first application process whose frontmost is true`)
	return err == nil && strings.EqualFold(strings.TrimSpace(out), cfg.AppName)
}

func sendPrompt(prompt string) error {
	// Set clipboard via pbcopy
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(prompt)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pbcopy failed: %w", err)
	}

	// Activate app
	if _, err := runOsascript(`tell application "` + cfg.AppName + `" to activate`); err != nil {
		return fmt.Errorf("activate failed: %w", err)
	}
	time.Sleep(500 * time.Millisecond)

	// Open chat panel (Cmd+shortcutKey)
	if err := runOsascriptMulti([]string{
		`tell application "System Events"`,
		`  keystroke "` + cfg.ShortcutKey + `" using {command down}`,
		`end tell`,
	}); err != nil {
		return fmt.Errorf("open chat failed: %w", err)
	}
	time.Sleep(1 * time.Second)

	// Select all + delete (clear input)
	if err := runOsascriptMulti([]string{
		`tell application "System Events"`,
		`  keystroke "a" using {command down}`,
		`  delay 0.2`,
		`  key code 51`,
		`end tell`,
	}); err != nil {
		return fmt.Errorf("clear input failed: %w", err)
	}
	time.Sleep(300 * time.Millisecond)

	// Paste (Cmd+V)
	if err := runOsascriptMulti([]string{
		`tell application "System Events"`,
		`  keystroke "v" using {command down}`,
		`end tell`,
	}); err != nil {
		return fmt.Errorf("paste failed: %w", err)
	}
	time.Sleep(300 * time.Millisecond)

	// Press Enter
	if err := runOsascriptMulti([]string{
		`tell application "System Events"`,
		`  key code 36`,
		`end tell`,
	}); err != nil {
		return fmt.Errorf("enter failed: %w", err)
	}

	return nil
}

// ─── GIT HELPERS ────────────────────────────────────────────────

func gitCmd(workspace string, args ...string) string {
	cmd := exec.Command("git", args...) // #nosec G204 — git is invoked with controlled arguments
	cmd.Dir = workspace
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func gitStatus(workspace string) string {
	return gitCmd(workspace, "status", "--porcelain")
}

func getWorkspaceChanges(workspace string) string {
	var parts []string

	// Status
	status := gitCmd(workspace, "status", "--short")
	if status != "" {
		parts = append(parts, "=== File Status ===\n"+status)
	}

	// Diff
	diff := gitCmd(workspace, "diff")
	if diff != "" {
		if len(diff) > 5000 {
			diff = diff[:5000] + "\n... (truncated)"
		}
		parts = append(parts, "=== Changes (diff) ===\n"+diff)
	}

	// Untracked files with preview
	untracked := gitCmd(workspace, "ls-files", "--others", "--exclude-standard")
	if untracked != "" {
		files := strings.Split(untracked, "\n")
		var previews []string
		for _, f := range files {
			if f == "" {
				continue
			}
			fullPath := filepath.Join(workspace, filepath.Clean(f))
			content, err := os.ReadFile(fullPath) // #nosec G304 — workspace path is user-configured
			if err != nil {
				previews = append(previews, fmt.Sprintf("--- %s --- (unreadable)", f))
				continue
			}
			preview := string(content)
			if len(preview) > 500 {
				preview = preview[:500] + "\n... (truncated)"
			}
			previews = append(previews, fmt.Sprintf("--- %s ---\n%s", f, preview))
		}
		parts = append(parts, "=== New Files ===\n"+strings.Join(previews, "\n\n"))
	}

	// Recent commits
	recentLog := gitCmd(workspace, "log", "--oneline", "-3", "--format=%h %s (%cr)")
	if recentLog != "" {
		parts = append(parts, "=== Recent Commits ===\n"+recentLog)
	}

	return strings.Join(parts, "\n\n")
}

func waitForChanges(workspace, initialStatus string, waitSecs int) waitResult {
	// Wait initial 5s for the app to start processing
	time.Sleep(5 * time.Second)

	deadline := time.Now().Add(time.Duration(waitSecs) * time.Second)
	for time.Now().Before(deadline) {
		current := gitStatus(workspace)
		if current != initialStatus {
			// Wait a bit more to let writes finish
			time.Sleep(3 * time.Second)
			return waitResult{
				Detected: true,
				Reason:   "File changes detected",
			}
		}
		time.Sleep(3 * time.Second)
	}

	// Timeout — gather diagnostic info
	appState := "not running"
	if isAppRunning() {
		if isAppFrontmost() {
			appState = "running and frontmost (may still be processing or waiting for input)"
		} else {
			appState = "running but not frontmost (another app took focus)"
		}
	}

	currentStatus := gitStatus(workspace)
	workspaceState := "clean (no uncommitted changes)"
	if currentStatus != "" {
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
