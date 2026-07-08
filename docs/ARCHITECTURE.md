# ClaudeDeck — Architecture

A single Go binary embeds the React UI, serves it on `localhost`, and does all the
system work the browser can't (reading `~/.claude`, `pgrep`/`lsof`/`ps`, `osascript`).

```
  ~/.claude files ──read──┐
  pgrep / lsof / ps ──────┤     ┌───────────────────────────┐        ┌──────────────────┐
  iTerm2 ◄── osascript ───┼────►│  Go binary                │◄─REST──│  React SPA       │
                          │     │  ingest · SQLite · serve  │  poll  │  (localhost:PORT)│
                          └────►└───────────────────────────┘        └──────────────────┘
```

## Packages
- **`internal/ingest`** — on startup, reads `~/.claude/history.jsonl` (prompts) and
  `~/.claude/projects/**/*.jsonl` (transcripts: model, git branch, message count, token
  usage), plus each session dir's optional `.claude/context/CURRENT_TASK.md` (task text).
  Dedupes by `sessionId` and upserts into SQLite.
- **`internal/store`** — SQLite via the pure-Go `modernc.org/sqlite` driver (no CGO, so
  the binary is fully static). Three tables (below).
- **`internal/live`** — computed per request, never stored: running sessions via
  `pgrep`+`lsof` (dedup by cwd) unioned with recent session activity; CPU%/RSS via `ps`;
  and pid→tty mapping used to focus the right terminal.
- **`internal/terminal`** — drives iTerm2 via AppleScript (focus a session by tty, resume
  via a new tab running `claude --resume <id>`). Kept behind small functions so other
  terminal emulators can be added later.
- **`internal/server`** — `net/http` REST API and serves the embedded SPA. The frontend
  polls every few seconds; status/CPU/mem are overlaid onto stored sessions at read time.
- **`web/frontend`** — React + TypeScript + Vite + Tailwind, built to `web/static/` and
  embedded into the binary via `//go:embed`.

## HTTP API
| Route | Purpose |
|---|---|
| `GET /api/sessions` | all sessions + live status/CPU/mem |
| `GET /api/stats` | dashboard totals |
| `GET /api/analytics` | daily activity, top projects, model/token breakdown |
| `POST /api/meta` | set favorite / tags / notes |
| `POST /api/action` | `focus` \| `resume` \| `kill` \| `reveal` |

## Data model (SQLite, `~/.claude/claude-deck.db`)
```sql
sessions(       -- a cache of data ingested from Claude Code's own files
  id TEXT PRIMARY KEY, cwd TEXT, project TEXT, created_at INTEGER, last_used_at INTEGER,
  duration_secs INTEGER, prompt_count INTEGER, message_count INTEGER, model TEXT,
  git_branch TEXT, first_prompt TEXT, tokens_in INTEGER, tokens_out INTEGER, current_task TEXT)

session_meta(   -- user data ClaudeDeck owns; a separate table so re-ingest never wipes it
  id TEXT PRIMARY KEY, favorite INTEGER, tags TEXT, notes TEXT)

daily(day TEXT PRIMARY KEY, prompts INTEGER)   -- activity heatmap
```
Live fields (`status`, `cpu`, `mem_mb`) are recomputed from the OS on each request; the DB
holds the durable/historical record. `session_meta` is the only original data ClaudeDeck
stores — everything in `sessions`/`daily` is derived from Claude Code's own files.

## Design notes
- **Why a local binary, not just a static site:** the dashboard needs process inspection
  and terminal control, which a browser can't do — so a local Go server is required.
- **Liveness is process-truth:** a session is "running" if a live `claude` process is in
  its directory, not by trusting a heartbeat — so idle-but-open sessions are detected.
- **Privacy:** binds to `localhost` only; no external calls, no telemetry; reads standard
  `~/.claude` data. Terminal actions happen only on explicit clicks.

## Platform
macOS + iTerm2 for the terminal actions (the rest of the dashboard is OS-agnostic in
principle). The terminal layer is isolated so other emulators/platforms can be added.
