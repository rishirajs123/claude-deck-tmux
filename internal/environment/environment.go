// Package environment derives skills and subagent usage purely from the files
// Claude Code already writes under ~/.claude — no hooks or extra setup needed.
// Skills come from "Skill" tool-use entries in session transcripts; subagent
// runs come from the per-run subagents/agent-*.jsonl files.
package environment

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type SkillStat struct {
	Name     string `json:"name"`
	Count    int    `json:"count"`
	LastUsed int64  `json:"last_used"` // epoch ms
}

type Count struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

type Stats struct {
	Skills        []SkillStat `json:"skills"`
	SkillRuns     int         `json:"skill_runs"`
	SkillCount    int         `json:"skill_count"`
	AgentRuns     int         `json:"agent_runs"`
	AgentByProj   []Count     `json:"agent_by_project"`
	AgentDaily    []Count     `json:"agent_daily"` // key = YYYY-MM-DD, chronological
	AgentProjects int         `json:"agent_projects"`
}

var skillMarker = []byte(`"Skill"`)

type skillLine struct {
	Timestamp string `json:"timestamp"`
	Message   struct {
		Content []struct {
			Type  string `json:"type"`
			Name  string `json:"name"`
			Input struct {
				Skill string `json:"skill"`
			} `json:"input"`
		} `json:"content"`
	} `json:"message"`
}

// Scan reads transcripts + subagent files and aggregates usage. Best-effort:
// unreadable files are skipped.
func Scan(claudeDir string) Stats {
	skills := scanSkills(claudeDir)
	st := Stats{Skills: skills, SkillCount: len(skills)}
	for _, s := range skills {
		st.SkillRuns += s.Count
	}
	scanAgents(claudeDir, &st)
	return st
}

func scanSkills(claudeDir string) []SkillStat {
	agg := map[string]*SkillStat{}
	files, _ := filepath.Glob(filepath.Join(claudeDir, "projects", "*", "*.jsonl"))
	for _, fp := range files {
		f, err := os.Open(fp)
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 1<<20), 64<<20)
		for sc.Scan() {
			b := sc.Bytes()
			if !bytes.Contains(b, skillMarker) {
				continue
			}
			var sl skillLine
			if json.Unmarshal(b, &sl) != nil {
				continue
			}
			ts := parseISO(sl.Timestamp)
			for _, c := range sl.Message.Content {
				if c.Type != "tool_use" || c.Name != "Skill" || c.Input.Skill == "" {
					continue
				}
				s := agg[c.Input.Skill]
				if s == nil {
					s = &SkillStat{Name: c.Input.Skill}
					agg[c.Input.Skill] = s
				}
				s.Count++
				if ts > s.LastUsed {
					s.LastUsed = ts
				}
			}
		}
		f.Close()
	}
	out := make([]SkillStat, 0, len(agg))
	for _, s := range agg {
		out = append(out, *s)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func scanAgents(claudeDir string, st *Stats) {
	files, _ := filepath.Glob(filepath.Join(claudeDir, "projects", "*", "*", "subagents", "*.jsonl"))
	byProj := map[string]int{}
	byDay := map[string]int{}
	for _, fp := range files {
		st.AgentRuns++
		proj, day := agentMeta(fp)
		if proj != "" {
			byProj[proj]++
		}
		if day != "" {
			byDay[day]++
		}
	}
	st.AgentProjects = len(byProj)
	st.AgentByProj = topCounts(byProj, 10)
	st.AgentDaily = sortedByKey(byDay)
}

func agentMeta(fp string) (proj, day string) {
	f, err := os.Open(fp)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 16<<20)
	if sc.Scan() {
		var r struct {
			Cwd       string `json:"cwd"`
			Timestamp string `json:"timestamp"`
		}
		if json.Unmarshal(sc.Bytes(), &r) == nil {
			if r.Cwd != "" {
				proj = filepath.Base(r.Cwd)
			}
			if ts := parseISO(r.Timestamp); ts > 0 {
				day = time.UnixMilli(ts).Format("2006-01-02")
			}
		}
	}
	return
}

func topCounts(m map[string]int, n int) []Count {
	out := make([]Count, 0, len(m))
	for k, v := range m {
		out = append(out, Count{Key: k, Count: v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Key < out[j].Key
	})
	if n > 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

func sortedByKey(m map[string]int) []Count {
	out := make([]Count, 0, len(m))
	for k, v := range m {
		out = append(out, Count{Key: k, Count: v})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func parseISO(s string) int64 {
	if s == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return 0
	}
	return t.UnixMilli()
}
