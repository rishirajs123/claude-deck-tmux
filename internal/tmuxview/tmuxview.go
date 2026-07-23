// Package tmuxview models the live tmux topology — every session, window and
// pane, on this machine and on machines reachable through ssh panes (tmux
// inside tmux, any depth) — and binds panes to the Claude Code sessions
// running in them.
//
// Binding needs no hooks, no pane tagging, no cooperation from Claude Code:
//   - a pane and a process share a tty (#{pane_tty} == the process's tty), and
//   - a resumed session names its uuid right in argv (`claude --resume <uuid>`);
//     fresh sessions are matched by pane title (Claude sets the terminal title
//     to the session title) and then by directory.
//
// Everything here works the same against a remote host: the same tmux formats
// and the same ps output are fetched over one ssh invocation, so a nested
// tmux on another machine is modeled — and controlled — exactly like the
// local one.
package tmuxview

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Pane is one tmux pane. IDs are tmux's stable identifiers (%n) — indexes and
// names are mutable attributes, never identity.
type Pane struct {
	ID      string `json:"id"`    // %3 — stable
	Index   int    `json:"index"` // display position, mutable
	PID     string `json:"pid"`   // pane's root process (the shell)
	TTY     string `json:"tty"`
	Cmd     string `json:"cmd"` // current foreground command
	Cwd     string `json:"cwd"`
	Title   string `json:"title"`
	Active  bool   `json:"active"`
	Claude  *Bind  `json:"claude,omitempty"` // set when a claude session runs here
	SSHDest string `json:"ssh_dest,omitempty"`
	// Nested is the tmux server reachable through this pane (an ssh pane whose
	// remote end runs tmux). The layer-within-layer case.
	Nested *Topology `json:"nested,omitempty"`
}

// Bind ties a pane to a Claude Code session, with how sure we are and why.
type Bind struct {
	SessionID string `json:"session_id,omitempty"`
	PID       string `json:"pid"`
	Bypass    bool   `json:"bypass"`
	Method    string `json:"method"` // argv | title | cwd-unique | cwd-newest | tty-only
}

type Window struct {
	ID     string `json:"id"`    // @2 — stable
	Index  int    `json:"index"` // mutable
	Name   string `json:"name"`
	Layout string `json:"layout"` // exact geometry, replayable via select-layout
	Active bool   `json:"active"`
	Panes  []Pane `json:"panes"`
}

type Session struct {
	ID       string   `json:"id"` // $0 — stable
	Name     string   `json:"name"`
	Attached bool     `json:"attached"`
	Windows  []Window `json:"windows"`
}

// Topology is one tmux server's full tree. Chain is how to reach it: empty for
// the local server, ["omarchy"] for tmux behind `ssh omarchy`, ["omarchy",
// "inner-host"] for a further hop. Exec on any Topology addresses that layer.
type Topology struct {
	Host     string    `json:"host"` // "local" or the ssh destination
	Chain    []string  `json:"chain"`
	Sessions []Session `json:"sessions"`
	Err      string    `json:"err,omitempty"` // probe failures, surfaced not hidden
}

// DeckSession is the slice of claude-deck's session record binding needs.
type DeckSession struct {
	ID         string
	Cwd        string
	Title      string // custom title, then ai_title — what Claude puts in the tty title
	CreatedAt  int64  // epoch ms
	LastUsedAt int64
}

// sep is a printable field separator: tmux ≥3.6 sanitizes control characters
// (a literal \t comes out as "_"), so tabs can't be trusted across versions.
// pane_title — the only free-text field — is last, and parsing uses SplitN, so
// a title containing sep can't shift other fields.
const sep = "|~|"

const paneFormat = "#{session_id}" + sep + "#{session_name}" + sep + "#{session_attached}" + sep +
	"#{window_id}" + sep + "#{window_index}" + sep + "#{window_name}" + sep + "#{window_layout}" + sep + "#{window_active}" + sep +
	"#{pane_id}" + sep + "#{pane_index}" + sep + "#{pane_pid}" + sep + "#{pane_tty}" + sep + "#{pane_current_command}" + sep + "#{pane_current_path}" + sep + "#{pane_active}" + sep + "#{pane_title}"

// probeCache remembers which ssh destinations have (or don't have) a reachable
// tmux, so the dashboard poll doesn't re-dial every few seconds.
var probeCache = struct {
	sync.Mutex
	m map[string]probeResult
}{m: map[string]probeResult{}}

type probeResult struct {
	top *Topology
	at  time.Time
	ok  bool
}

const (
	probeOKTTL   = 20 * time.Second // re-read a live remote this often
	probeFailTTL = 5 * time.Minute  // don't hammer unreachable hosts
)

// Snapshot reads the local tmux server and, when followSSH is true, descends
// into every ssh pane that leads to another tmux (recursively, bounded).
// deck sessions power the claude bindings; pass nil to skip binding.
func Snapshot(deck []DeckSession, followSSH bool) *Topology {
	top := readLayer(nil, deck)
	if followSSH && top.Err == "" {
		descend(top, deck, 3)
	}
	return top
}

// descend probes ssh panes of t for nested tmux servers, depth layers deep.
func descend(t *Topology, deck []DeckSession, depth int) {
	if depth == 0 {
		return
	}
	for si := range t.Sessions {
		for wi := range t.Sessions[si].Windows {
			for pi := range t.Sessions[si].Windows[wi].Panes {
				p := &t.Sessions[si].Windows[wi].Panes[pi]
				if p.Cmd != "ssh" {
					continue
				}
				dest := sshDest(p.PID)
				if dest == "" {
					continue
				}
				p.SSHDest = dest
				chain := append(append([]string{}, t.Chain...), dest)
				if nested := probeLayer(chain, deck); nested != nil {
					p.Nested = nested
					descend(nested, deck, depth-1)
				}
			}
		}
	}
}

// probeLayer returns the topology behind an ssh chain, using the cache.
func probeLayer(chain []string, deck []DeckSession) *Topology {
	key := strings.Join(chain, "→")
	probeCache.Lock()
	if r, ok := probeCache.m[key]; ok {
		ttl := probeOKTTL
		if !r.ok {
			ttl = probeFailTTL
		}
		if time.Since(r.at) < ttl {
			probeCache.Unlock()
			return r.top
		}
	}
	probeCache.Unlock()

	top := readLayer(chain, deck)
	ok := top.Err == ""
	if !ok {
		top = nil // an unreachable/tmux-less host is not a layer
	}
	probeCache.Lock()
	probeCache.m[key] = probeResult{top: top, at: time.Now(), ok: ok}
	probeCache.Unlock()
	return top
}

// readLayer fetches one tmux server's tree + claude processes in a single
// (possibly remote) shell invocation and assembles the topology.
func readLayer(chain []string, deck []DeckSession) *Topology {
	host := "local"
	if len(chain) > 0 {
		host = chain[len(chain)-1]
	}
	t := &Topology{Host: host, Chain: chain}

	// One round-trip: panes, then a marker, then every claude-ish process.
	script := fmt.Sprintf(`tmux list-panes -a -F '%s' 2>&1; echo '===PS==='; ps -Ao pid=,tty=,stat=,args= | grep -w claude | grep -v grep`, paneFormat)
	out, err := runShell(chain, script)
	if err != nil && !strings.Contains(out, "===PS===") {
		t.Err = strings.TrimSpace(firstLine(out) + " " + err.Error())
		return t
	}
	tmuxPart, psPart, _ := strings.Cut(out, "===PS===")
	if strings.Contains(tmuxPart, "no server running") || strings.Contains(tmuxPart, "error connecting") {
		t.Err = "no tmux server"
		return t
	}

	claudeByTTY := parseClaudeProcs(psPart)

	// list-panes -a emits panes already grouped by session then window, so the
	// tree assembles in one ordered pass.
	var cur *Session
	var curWin *Window
	for _, line := range strings.Split(strings.TrimSpace(tmuxPart), "\n") {
		f := strings.SplitN(line, sep, 16)
		if len(f) < 16 {
			continue
		}
		if cur == nil || cur.ID != f[0] {
			t.Sessions = append(t.Sessions, Session{ID: f[0], Name: f[1], Attached: f[2] != "0"})
			cur = &t.Sessions[len(t.Sessions)-1]
			curWin = nil
		}
		if curWin == nil || curWin.ID != f[3] {
			cur.Windows = append(cur.Windows, Window{
				ID: f[3], Index: atoi(f[4]), Name: f[5], Layout: f[6], Active: f[7] == "1",
			})
			curWin = &cur.Windows[len(cur.Windows)-1]
		}
		pane := Pane{
			ID: f[8], Index: atoi(f[9]), PID: f[10], TTY: f[11],
			Cmd: f[12], Cwd: f[13], Active: f[14] == "1", Title: f[15],
		}
		if cp, ok := claudeByTTY[strings.TrimPrefix(pane.TTY, "/dev/")]; ok {
			pane.Claude = bindPane(pane, cp, deck)
		}
		curWin.Panes = append(curWin.Panes, pane)
	}
	return t
}

type claudeProc struct {
	pid    string
	args   string
	bypass bool
	resume string // session uuid from `--resume <uuid>`, if any
}

var resumeRe = regexp.MustCompile(`--resume[= ]+([0-9a-f-]{36})`)

// parseClaudeProcs filters ps output to live interactive claude sessions
// (same rules as internal/live) and indexes them by tty.
func parseClaudeProcs(ps string) map[string]claudeProc {
	m := map[string]claudeProc{}
	for _, line := range strings.Split(ps, "\n") {
		f := strings.Fields(line)
		if len(f) < 4 || f[1] == "??" || f[1] == "?" || strings.HasPrefix(f[2], "T") {
			continue
		}
		if strings.Contains(line, "claude-deck") || strings.Contains(line, "Claude.app") {
			continue
		}
		p := claudeProc{pid: f[0], args: strings.Join(f[3:], " ")}
		p.bypass = strings.Contains(p.args, "--dangerously-skip-permissions") ||
			strings.Contains(p.args, "bypassPermissions")
		if mm := resumeRe.FindStringSubmatch(p.args); mm != nil {
			p.resume = mm[1]
		}
		m[f[1]] = p
	}
	return m
}

// bindPane decides which deck session runs in this pane. Methods, strongest
// first: argv uuid → pane-title match → only session in this cwd → newest in
// this cwd. Remote layers have no deck records, so argv or tty-only.
func bindPane(p Pane, cp claudeProc, deck []DeckSession) *Bind {
	b := &Bind{PID: cp.pid, Bypass: cp.bypass}
	if cp.resume != "" {
		b.SessionID, b.Method = cp.resume, "argv"
		return b
	}
	var inCwd []DeckSession
	for _, d := range deck {
		if d.Cwd == p.Cwd {
			inCwd = append(inCwd, d)
		}
	}
	// Claude sets the terminal title to the session title, prefixed by a
	// spinner glyph — substring match is deliberate.
	title := strings.TrimSpace(p.Title)
	for _, d := range inCwd {
		if d.Title != "" && strings.Contains(title, d.Title) {
			b.SessionID, b.Method = d.ID, "title"
			return b
		}
	}
	switch len(inCwd) {
	case 0:
		b.Method = "tty-only"
	case 1:
		b.SessionID, b.Method = inCwd[0].ID, "cwd-unique"
	default:
		newest := inCwd[0]
		for _, d := range inCwd[1:] {
			if d.LastUsedAt > newest.LastUsedAt {
				newest = d
			}
		}
		b.SessionID, b.Method = newest.ID, "cwd-newest"
	}
	return b
}

// Exec runs a tmux command against the layer addressed by chain — locally when
// chain is empty, else through ssh hop(s). This is the single control path for
// "change anything at any layer": kill a window two machines deep, rename a
// remote session, send keys into a nested pane.
func Exec(chain []string, args ...string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("no tmux arguments")
	}
	return runShell(chain, "tmux "+shellJoin(args))
}

// runShell executes script locally or through the ssh chain.
func runShell(chain []string, script string) (string, error) {
	cmd := script
	for i := len(chain) - 1; i >= 0; i-- {
		cmd = "ssh -o BatchMode=yes -o ConnectTimeout=4 " + chain[i] + " " + shellJoin([]string{cmd})
	}
	out, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	return string(out), err
}

// sshDest extracts the destination of the ssh process running under pane pid.
func sshDest(panePID string) string {
	out, err := exec.Command("/bin/sh", "-c",
		"ps -o args= -p $(pgrep -P "+panePID+" | tr '\\n' ',' | sed 's/,$//') 2>/dev/null").Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) == 0 || f[0] != "ssh" {
			continue
		}
		// last argument that isn't a flag or a flag's value: the destination
		dest := ""
		skip := false
		for _, a := range f[1:] {
			if skip {
				skip = false
				continue
			}
			if strings.HasPrefix(a, "-") {
				// flags that consume a value
				if strings.ContainsAny(strings.TrimPrefix(a, "-"), "bcDEeFIiJLlmOopQRSWw") && len(a) == 2 {
					skip = true
				}
				continue
			}
			dest = a
			break // first non-flag arg is the destination; rest is remote cmd
		}
		return dest
	}
	return ""
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	return n
}

// shellJoin quotes args for /bin/sh.
func shellJoin(args []string) string {
	q := make([]string, len(args))
	for i, a := range args {
		q[i] = "'" + strings.ReplaceAll(a, "'", `'\''`) + "'"
	}
	return strings.Join(q, " ")
}
