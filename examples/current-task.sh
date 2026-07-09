#!/bin/bash
# ClaudeDeck — optional "current task" SessionStart hook.
#
# Creates .claude/context/CURRENT_TASK.md in each session's working directory so
# the ClaudeDeck dashboard can show a task summary per session. Edit the line
# under "## Task" (or have Claude keep it updated); ClaudeDeck reads that line.
#
# Install with ./install-task-hook.sh, or copy this to ~/.claude/hooks/ and add a
# SessionStart hook pointing at it (see ../docs/TASK_TRACKING.md).
set -uo pipefail

INPUT=$(cat 2>/dev/null || echo "")
CWD=$(printf '%s' "$INPUT" | jq -r '.cwd // empty' 2>/dev/null)
[ -z "$CWD" ] && CWD="$PWD"

FILE="$CWD/.claude/context/CURRENT_TASK.md"
mkdir -p "$(dirname "$FILE")" 2>/dev/null || exit 0

if [ ! -f "$FILE" ]; then
  cat > "$FILE" <<'EOF'
# Current Task

## Task
<what this session is working on>

## Next step
<the next action>
EOF
fi
exit 0
