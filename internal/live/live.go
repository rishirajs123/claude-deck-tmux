package live

import (
	"os/exec"
	"strconv"
	"strings"
)

// ClaudeProcs maps each working directory to the pids of the live interactive
// `claude` sessions running in it. A real session has a controlling tty; this
// ignores Claude's background daemon/spare/pty-host workers (tty "??"), stopped
// (Ctrl-Z'd) processes, the Claude desktop app, and the codebase-memory /
// claude-deck binaries.
func ClaudeProcs() map[string][]string {
	m := map[string][]string{}
	// Enumerate every process — `pgrep -f claude` unreliably misses live sessions.
	out, err := exec.Command("ps", "-Ao", "pid=,tty=,stat=,command=").Output()
	if err != nil {
		return m
	}
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) < 4 || !strings.Contains(line, "claude") {
			continue
		}
		if f[1] == "??" || f[1] == "?" { // no controlling tty: a daemon, not a session
			continue
		}
		if strings.HasPrefix(f[2], "T") { // stopped/suspended: a Ctrl-Z'd claude
			continue // lingers for days but is not a live session
		}
		if strings.Contains(line, "codebase-memory") || strings.Contains(line, "claude-deck") ||
			strings.Contains(line, "Applications/Claude.app") || strings.Contains(line, "Claude Helper") {
			continue
		}
		lo, _ := exec.Command("lsof", "-a", "-p", f[0], "-d", "cwd", "-Fn").Output()
		for _, l := range strings.Split(string(lo), "\n") {
			if strings.HasPrefix(l, "n/") {
				m[strings.TrimPrefix(l, "n")] = append(m[strings.TrimPrefix(l, "n")], f[0])
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

// UsageForPids sums %CPU and resident memory (MB) across the given pids.
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
