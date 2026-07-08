#!/bin/bash
# Remove the ClaudeDeck LaunchAgent.
PLIST="$HOME/Library/LaunchAgents/com.claudedeck.agent.plist"
launchctl unload "$PLIST" 2>/dev/null || true
rm -f "$PLIST"
echo "✓ removed com.claudedeck.agent (existing data in ~/.claude/claude-deck.db is untouched)"
