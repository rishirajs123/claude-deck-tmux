package server

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/tachodril/claude-deck/internal/ingest"
	"github.com/tachodril/claude-deck/internal/live"
	"github.com/tachodril/claude-deck/internal/model"
	"github.com/tachodril/claude-deck/internal/store"
	"github.com/tachodril/claude-deck/internal/terminal"
	"github.com/tachodril/claude-deck/web"
)

type Server struct {
	st         *store.Store
	claudeDir  string
	mu         sync.Mutex
	lastIngest time.Time
	ingesting  bool
}

func New(st *store.Store, claudeDir string) *Server {
	return &Server{st: st, claudeDir: claudeDir, lastIngest: time.Now()}
}

// maybeReingest refreshes the DB from ~/.claude at most every 15s, in the
// background, so stored fields (last_used_at, prompts, tokens) stay current
// while the dashboard is open — and costs nothing when nobody's watching.
func (s *Server) maybeReingest() {
	s.mu.Lock()
	if s.ingesting || time.Since(s.lastIngest) < 15*time.Second {
		s.mu.Unlock()
		return
	}
	s.ingesting = true
	s.mu.Unlock()
	go func() {
		ingest.Run(s.claudeDir, s.st)
		s.mu.Lock()
		s.lastIngest = time.Now()
		s.ingesting = false
		s.mu.Unlock()
	}()
}

func (s *Server) Listen(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/sessions", s.handleSessions)
	mux.HandleFunc("/api/stats", s.handleStats)
	mux.HandleFunc("/api/analytics", s.handleAnalytics)
	mux.HandleFunc("/api/meta", s.handleMeta)
	mux.HandleFunc("/api/action", s.handleAction)
	sub, _ := fs.Sub(web.FS, "static")
	mux.Handle("/", http.FileServer(http.FS(sub)))
	return http.ListenAndServe(addr, mux)
}

// withStatus loads sessions and overlays live status: a cwd with a running
// claude process marks its most-recently-used session as "running".
func (s *Server) withStatus() ([]model.Session, error) {
	s.maybeReingest()
	sessions, err := s.st.All()
	if err != nil {
		return nil, err
	}
	procs := live.ClaudeProcs()
	running := live.RegistryCwds()
	for cwd := range procs {
		running[cwd] = true
	}
	newest := map[string]int64{}
	for _, ss := range sessions {
		if running[ss.Cwd] && ss.LastUsedAt > newest[ss.Cwd] {
			newest[ss.Cwd] = ss.LastUsedAt
		}
	}
	usageCache := map[string][2]float64{}
	for i := range sessions {
		s := &sessions[i]
		if running[s.Cwd] && s.LastUsedAt == newest[s.Cwd] {
			s.Status = "running"
			if u, ok := usageCache[s.Cwd]; ok {
				s.CPU, s.MemMB = u[0], u[1]
			} else if pids := procs[s.Cwd]; len(pids) > 0 {
				s.CPU, s.MemMB = live.UsageForPids(pids)
				usageCache[s.Cwd] = [2]float64{s.CPU, s.MemMB}
			}
		} else {
			s.Status = "ended"
		}
	}
	return sessions, nil
}

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := s.withStatus()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, sessions)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	sessions, err := s.withStatus()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	stats := struct {
		Total     int   `json:"total"`
		Running   int   `json:"running"`
		Prompts   int   `json:"prompts"`
		Projects  int   `json:"projects"`
		TokensIn  int64 `json:"tokens_in"`
		TokensOut int64 `json:"tokens_out"`
	}{}
	projs := map[string]bool{}
	for _, ss := range sessions {
		stats.Total++
		stats.Prompts += ss.PromptCount
		stats.TokensIn += ss.TokensIn
		stats.TokensOut += ss.TokensOut
		if ss.Status == "running" {
			stats.Running++
		}
		if ss.Cwd != "" {
			projs[ss.Cwd] = true
		}
	}
	stats.Projects = len(projs)
	writeJSON(w, stats)
}

func (s *Server) handleAnalytics(w http.ResponseWriter, r *http.Request) {
	s.maybeReingest()
	sessions, err := s.st.All()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	days, _ := s.st.Daily()

	type agg struct {
		Key      string `json:"key"`
		Sessions int    `json:"sessions"`
		Prompts  int    `json:"prompts"`
		Tokens   int64  `json:"tokens"`
	}
	byProj := map[string]*agg{}
	byModel := map[string]*agg{}
	bump := func(m map[string]*agg, key string, s model.Session) {
		if key == "" {
			return
		}
		a := m[key]
		if a == nil {
			a = &agg{Key: key}
			m[key] = a
		}
		a.Sessions++
		a.Prompts += s.PromptCount
		a.Tokens += s.TokensIn + s.TokensOut
	}
	for _, ss := range sessions {
		bump(byProj, ss.Project, ss)
		model := ss.Model
		if model == "" {
			model = "unknown"
		}
		bump(byModel, model, ss)
	}
	topN := func(m map[string]*agg, sortByTokens bool, n int) []agg {
		list := make([]agg, 0, len(m))
		for _, a := range m {
			list = append(list, *a)
		}
		sort.Slice(list, func(i, j int) bool {
			if sortByTokens {
				return list[i].Tokens > list[j].Tokens
			}
			return list[i].Prompts > list[j].Prompts
		})
		if n > 0 && len(list) > n {
			list = list[:n]
		}
		return list
	}
	writeJSON(w, map[string]any{
		"daily":       days,
		"topProjects": topN(byProj, false, 12),
		"models":      topN(byModel, true, 0),
	})
}

func (s *Server) handleMeta(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	var req struct {
		ID       string `json:"id"`
		Favorite bool   `json:"favorite"`
		Tags     string `json:"tags"`
		Notes    string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
		http.Error(w, "bad request", 400)
		return
	}
	if err := s.st.SetMeta(req.ID, req.Favorite, req.Tags, req.Notes); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) handleAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	var req struct{ Action, ID, Cwd, Text string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	var err error
	switch req.Action {
	case "focus":
		err = terminal.Focus(req.Cwd)
	case "resume":
		err = terminal.Resume(req.Cwd, req.ID)
	case "new":
		err = terminal.NewSession(req.Cwd)
	case "kill":
		err = terminal.Kill(req.Cwd)
	case "reveal":
		err = terminal.Reveal(req.Cwd)
	case "send":
		err = terminal.SendText(req.Cwd, req.Text)
	default:
		err = fmt.Errorf("unknown action %q", req.Action)
	}
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
