package live

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ClaudeProcs maps each working directory to the pids of the live `claude`
// processes running in it. Child processes inherit the session cwd. Excludes
// the codebase-memory-mcp and claude-deck binaries.
func ClaudeProcs() map[string][]string {
	m := map[string][]string{}
	out, err := exec.Command("pgrep", "-f", "claude").Output()
	if err != nil {
		return m
	}
	for _, pid := range strings.Fields(string(out)) {
		cmd, _ := exec.Command("ps", "-o", "command=", "-p", pid).Output()
		c := string(cmd)
		if strings.Contains(c, "codebase-memory") || strings.Contains(c, "claude-deck") {
			continue
		}
		lo, err := exec.Command("lsof", "-a", "-p", pid, "-d", "cwd", "-Fn").Output()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(lo), "\n") {
			if strings.HasPrefix(line, "n/") {
				cwd := strings.TrimPrefix(line, "n")
				m[cwd] = append(m[cwd], pid)
			}
		}
	}
	return m
}

// TtyForCwd returns the controlling tty (e.g. /dev/ttys002) of a claude process
// running in cwd, or "" if none.
func TtyForCwd(cwd string) string {
	for _, pid := range ClaudeProcs()[cwd] {
		out, err := exec.Command("ps", "-o", "tty=", "-p", pid).Output()
		if err != nil {
			continue
		}
		t := strings.TrimSpace(string(out))
		if t != "" && t != "??" {
			return "/dev/" + t
		}
	}
	return ""
}

// RunningCwds returns the set of directories with a live session: claude
// processes (via ClaudeProcs) unioned with registry entries touched in the last
// 15 min (catches this session, which appears only as transient shell wrappers).
func RunningCwds() map[string]bool {
	set := RegistryCwds()
	for cwd := range ClaudeProcs() {
		set[cwd] = true
	}
	return set
}

// RegistryCwds returns cwds of session-registry entries touched in the last 15 min.
func RegistryCwds() map[string]bool {
	set := map[string]bool{}
	home, err := os.UserHomeDir()
	if err != nil {
		return set
	}
	dir := filepath.Join(home, ".claude", "session-registry")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return set
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		info, err := e.Info()
		if err != nil || time.Since(info.ModTime()) > 15*time.Minute {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var r struct {
			Cwd string `json:"cwd"`
		}
		if json.Unmarshal(data, &r) == nil && r.Cwd != "" {
			set[r.Cwd] = true
		}
	}
	return set
}

// UsageForPids sums %CPU and resident memory (MB) across the given pids.
// Bypassed reports whether any of the pids was launched with permission checks
// bypassed (--dangerously-skip-permissions or --permission-mode bypassPermissions).
// bypass is a startup-only mode, so the process args are authoritative.
func Bypassed(pids []string) bool {
	if len(pids) == 0 {
		return false
	}
	out, err := exec.Command("ps", "-o", "args=", "-p", strings.Join(pids, ",")).Output()
	if err != nil {
		return false
	}
	s := string(out)
	return strings.Contains(s, "--dangerously-skip-permissions") || strings.Contains(s, "bypassPermissions")
}

func UsageForPids(pids []string) (cpu, memMB float64) {
	if len(pids) == 0 {
		return 0, 0
	}
	out, err := exec.Command("ps", "-o", "%cpu=,rss=", "-p", strings.Join(pids, ",")).Output()
	if err != nil {
		return 0, 0
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		if c, e := strconv.ParseFloat(f[0], 64); e == nil {
			cpu += c
		}
		if r, e := strconv.ParseFloat(f[1], 64); e == nil {
			memMB += r / 1024
		}
	}
	return cpu, memMB
}
