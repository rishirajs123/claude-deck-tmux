package store

import (
	"database/sql"

	"github.com/tachodril/claude-deck/internal/model"
	_ "modernc.org/sqlite"
)

type Store struct{ db *sql.DB }

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	// WAL lets reads proceed during the background re-ingest write.
	if _, err := db.Exec(`PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000;`); err != nil {
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}
	// Migrations for DBs created before a column existed (ADD COLUMN is a no-op
	// error if it's already there, which is fine).
	for _, col := range []string{"title TEXT DEFAULT ''", "ai_title TEXT DEFAULT ''"} {
		db.Exec(`ALTER TABLE sessions ADD COLUMN ` + col)
	}
	return &Store{db: db}, nil
}

const schema = `
CREATE TABLE IF NOT EXISTS sessions (
  id            TEXT PRIMARY KEY,
  cwd           TEXT,
  project       TEXT,
  created_at    INTEGER,
  last_used_at  INTEGER,
  duration_secs INTEGER,
  prompt_count  INTEGER,
  message_count INTEGER,
  model         TEXT,
  git_branch    TEXT,
  first_prompt  TEXT,
  tokens_in     INTEGER,
  tokens_out    INTEGER,
  current_task  TEXT,
  title         TEXT DEFAULT '',
  ai_title      TEXT DEFAULT ''
);
CREATE TABLE IF NOT EXISTS session_meta (
  id       TEXT PRIMARY KEY,
  favorite INTEGER DEFAULT 0,
  tags     TEXT DEFAULT '',
  notes    TEXT DEFAULT ''
);
CREATE TABLE IF NOT EXISTS daily (
  day     TEXT PRIMARY KEY,
  prompts INTEGER
);`

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) UpsertAll(sessions []model.Session) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`
INSERT INTO sessions (id,cwd,project,created_at,last_used_at,duration_secs,prompt_count,message_count,model,git_branch,first_prompt,tokens_in,tokens_out,current_task,title,ai_title)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  cwd=excluded.cwd, project=excluded.project, created_at=excluded.created_at,
  last_used_at=excluded.last_used_at, duration_secs=excluded.duration_secs,
  prompt_count=excluded.prompt_count, message_count=excluded.message_count,
  model=excluded.model, git_branch=excluded.git_branch, first_prompt=excluded.first_prompt,
  tokens_in=excluded.tokens_in, tokens_out=excluded.tokens_out, current_task=excluded.current_task,
  title=excluded.title, ai_title=excluded.ai_title`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, m := range sessions {
		if _, err := stmt.Exec(m.ID, m.Cwd, m.Project, m.CreatedAt, m.LastUsedAt, m.DurationSecs,
			m.PromptCount, m.MessageCount, m.Model, m.GitBranch, m.FirstPrompt, m.TokensIn, m.TokensOut, m.CurrentTask, m.Title, m.AiTitle); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// SetDaily replaces the daily activity table.
func (s *Store) SetDaily(days map[string]int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM daily`); err != nil {
		tx.Rollback()
		return err
	}
	stmt, _ := tx.Prepare(`INSERT INTO daily (day,prompts) VALUES (?,?)`)
	defer stmt.Close()
	for d, n := range days {
		if _, err := stmt.Exec(d, n); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) Daily() ([]model.Day, error) {
	rows, err := s.db.Query(`SELECT day,prompts FROM daily ORDER BY day`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Day
	for rows.Next() {
		var d model.Day
		if err := rows.Scan(&d.Day, &d.Prompts); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// SetMeta upserts user metadata (favorite/tags/notes) for a session.
func (s *Store) SetMeta(id string, favorite bool, tags, notes string) error {
	fav := 0
	if favorite {
		fav = 1
	}
	_, err := s.db.Exec(`
INSERT INTO session_meta (id,favorite,tags,notes) VALUES (?,?,?,?)
ON CONFLICT(id) DO UPDATE SET favorite=excluded.favorite, tags=excluded.tags, notes=excluded.notes`,
		id, fav, tags, notes)
	return err
}

func (s *Store) All() ([]model.Session, error) {
	rows, err := s.db.Query(`
SELECT s.id,s.cwd,s.project,s.created_at,s.last_used_at,s.duration_secs,s.prompt_count,s.message_count,
       s.model,s.git_branch,s.first_prompt,s.tokens_in,s.tokens_out,s.current_task,
       COALESCE(s.title,''),COALESCE(s.ai_title,''),
       COALESCE(m.favorite,0),COALESCE(m.tags,''),COALESCE(m.notes,'')
FROM sessions s LEFT JOIN session_meta m ON m.id=s.id
ORDER BY s.last_used_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Session
	for rows.Next() {
		var m model.Session
		var fav int
		if err := rows.Scan(&m.ID, &m.Cwd, &m.Project, &m.CreatedAt, &m.LastUsedAt, &m.DurationSecs,
			&m.PromptCount, &m.MessageCount, &m.Model, &m.GitBranch, &m.FirstPrompt, &m.TokensIn, &m.TokensOut, &m.CurrentTask,
			&m.Title, &m.AiTitle,
			&fav, &m.Tags, &m.Notes); err != nil {
			return nil, err
		}
		m.Favorite = fav == 1
		out = append(out, m)
	}
	return out, rows.Err()
}
