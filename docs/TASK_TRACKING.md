# The Task column (optional)

ClaudeDeck shows a **Task** for each session — a one-line summary of what that session is working on. It's optional enrichment:

- **If present**, ClaudeDeck reads it from `<session-cwd>/.claude/context/CURRENT_TASK.md`.
- **If absent**, the Task column falls back to the session's first prompt.

So out of the box you'll see first prompts. To get real, curated task summaries, add the tiny convention below. Nothing else about ClaudeDeck depends on it.

## What ClaudeDeck reads
For each session's working directory, it opens `.claude/context/CURRENT_TASK.md` and takes the **first non-empty line under a `## Task` heading** (ignoring blockquotes and `<placeholder>` lines). That's the entire contract:

```markdown
# Current Task

## Task
Add idempotency keys to the webhook handler
```

## Option A — do it by hand
Drop that file into any project's `.claude/context/` and edit the `## Task` line whenever you switch tasks. Simplest, zero setup.

## Option B — auto-create it with a SessionStart hook
Have Claude Code create the file automatically at the start of every session, so you only edit the one line.

1. Save this as `~/.claude/hooks/current-task.sh` and `chmod +x` it:

```bash
#!/bin/bash
# Bootstraps .claude/context/CURRENT_TASK.md so ClaudeDeck's Task column has data.
set -uo pipefail
INPUT=$(cat 2>/dev/null || echo "")
CWD=$(printf '%s' "$INPUT" | jq -r '.cwd // empty' 2>/dev/null); [ -z "$CWD" ] && CWD="$PWD"
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
```

2. Register it in `~/.claude/settings.json`:

```json
{
  "hooks": {
    "SessionStart": [
      { "matcher": "", "hooks": [ { "type": "command", "command": "~/.claude/hooks/current-task.sh" } ] }
    ]
  }
}
```

3. (Optional) Keep the line accurate automatically — add to `~/.claude/CLAUDE.md`:
   > Keep `.claude/context/CURRENT_TASK.md`'s `## Task` line current: a one-line summary of the task in progress. Update it when the task changes; don't edit on every message.

## Keep it out of git
It's local, per-directory context — don't commit it:
```bash
echo '.claude/context/' >> .git/info/exclude   # per-clone, not tracked
```

That's it. Next time you open ClaudeDeck, sessions in directories that have this file will show their task instead of the first prompt.

> This is the minimal piece ClaudeDeck needs. It's a small slice of a broader "session context" pattern (durable notes, plans, cross-session awareness) — but none of that is required for the Task column.
