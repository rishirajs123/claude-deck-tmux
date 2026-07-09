# Examples

Optional add-ons for ClaudeDeck. Nothing here is required — the dashboard works
without them.

## Task column (`current-task.sh`)
Populates the dashboard's **Task** column with a real, curated summary per session
instead of the first prompt.

**One-command install** (requires `jq`):
```bash
./install-task-hook.sh
```
This copies `current-task.sh` to `~/.claude/hooks/` and registers a `SessionStart`
hook in `~/.claude/settings.json` (idempotent — safe to re-run).

After that, every new Claude Code session creates `.claude/context/CURRENT_TASK.md`
in its directory. Edit the line under `## Task` (or tell Claude to keep it current);
ClaudeDeck reads that line. Keep it out of git with
`echo '.claude/context/' >> .git/info/exclude`.

See [`../docs/TASK_TRACKING.md`](../docs/TASK_TRACKING.md) for the manual setup and
the exact contract ClaudeDeck reads.
