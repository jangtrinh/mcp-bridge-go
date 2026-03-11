#!/bin/bash
# ======================================================
# Paperclip → Gemini CLI Adapter
# ======================================================
# Paperclip's Claude adapter hardcodes Claude-specific flags
# (--print, --verbose, --max-turns, --add-dir, etc.)
# This wrapper IGNORES all positional args and reads
# the prompt from stdin (which the adapter provides),
# then forwards it to Gemini CLI instead.
#
# Usage:
#   1. Set this script as the agent command in Paperclip
#   2. Gemini CLI must have MCP Bridge configured in ~/.gemini/settings.json
#   3. Make sure this script is executable: chmod +x scripts/paperclip-adapter.sh
# ======================================================

# Read prompt from stdin (Paperclip sends it this way)
PROMPT=$(cat)

# Fallback if stdin is empty
if [ -z "$PROMPT" ]; then
  PROMPT="Continue your Paperclip work. Check for pending tasks."
fi

# Log for debugging (visible in Paperclip run logs)
echo "[paperclip-adapter] Received prompt (${#PROMPT} chars)" >&2

# Call Gemini CLI — ignore ALL args passed by the Claude adapter
exec gemini --prompt "$PROMPT" --yolo --output-format stream-json
