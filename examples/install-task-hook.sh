#!/bin/bash
# Install the optional ClaudeDeck "current task" SessionStart hook.
# Idempotent: safe to re-run. Requires jq.
set -euo pipefail

command -v jq >/dev/null 2>&1 || { echo "error: jq is required (brew install jq)"; exit 1; }

DIR="$(cd "$(dirname "$0")" && pwd)"
HOOK="$HOME/.claude/hooks/current-task.sh"
SETTINGS="$HOME/.claude/settings.json"
CMD="~/.claude/hooks/current-task.sh"

mkdir -p "$HOME/.claude/hooks"
install -m 0755 "$DIR/current-task.sh" "$HOOK"

[ -f "$SETTINGS" ] || echo '{}' > "$SETTINGS"
tmp="$(mktemp)"
jq --arg cmd "$CMD" '
  .hooks //= {} | .hooks.SessionStart //= []
  | if any(.hooks.SessionStart[]?.hooks[]?; .command == $cmd)
    then .
    else .hooks.SessionStart += [{matcher:"",hooks:[{type:"command",command:$cmd}]}]
    end
' "$SETTINGS" > "$tmp" && mv "$tmp" "$SETTINGS"

echo "✓ installed $HOOK and registered the SessionStart hook."
echo "  New Claude Code sessions will create .claude/context/CURRENT_TASK.md."
echo "  Edit the line under '## Task' and ClaudeDeck shows it."
echo "  Keep it out of git:  echo '.claude/context/' >> .git/info/exclude"
