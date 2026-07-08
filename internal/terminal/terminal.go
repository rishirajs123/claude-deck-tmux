// Package terminal drives iTerm2 via AppleScript to focus, resume, or start
// Claude Code sessions. macOS + iTerm2 only for now (behind these functions so
// other emulators can be added later).
package terminal

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/tachodril/claude-deck/internal/live"
)

func osa(script string) (string, error) {
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	s := strings.TrimSpace(string(out))
	if err != nil {
		return s, fmt.Errorf("osascript: %v: %s", err, s)
	}
	return s, nil
}

// Focus brings the iTerm2 tab of the session running in cwd to the front.
func Focus(cwd string) error {
	tty := live.TtyForCwd(cwd)
	if tty == "" {
		return fmt.Errorf("no running session found in %s", cwd)
	}
	script := fmt.Sprintf(`tell application "iTerm2"
  repeat with w in windows
    repeat with t in tabs of w
      repeat with s in sessions of t
        if tty of s is %q then
          select s
          select t
          try
            set index of w to 1
          end try
          activate
          return "ok"
        end if
      end repeat
    end repeat
  end repeat
  return "notfound"
end tell`, tty)
	out, err := osa(script)
	if err != nil {
		return err
	}
	if out == "notfound" {
		return fmt.Errorf("session tty %s not found among iTerm2 tabs", tty)
	}
	return nil
}

// Resume opens a new iTerm2 tab in cwd and resumes the given session id.
func Resume(cwd, id string) error {
	return openTab(shellCmd(cwd, "claude --resume "+shQuote(id)))
}

// NewSession opens a new iTerm2 tab in cwd and starts a fresh claude session.
func NewSession(cwd string) error {
	return openTab(shellCmd(cwd, "claude"))
}

// Kill terminates the claude process(es) running in cwd.
func Kill(cwd string) error {
	pids := live.ClaudeProcs()[cwd]
	if len(pids) == 0 {
		return fmt.Errorf("no running session in %s", cwd)
	}
	for _, pid := range pids {
		_ = exec.Command("kill", pid).Run()
	}
	return nil
}

// Reveal opens cwd in Finder.
func Reveal(cwd string) error { return exec.Command("open", cwd).Run() }

func shQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

func shellCmd(cwd, cmd string) string { return "cd " + shQuote(cwd) + " && " + cmd }

func openTab(cmd string) error {
	esc := strings.ReplaceAll(cmd, `\`, `\\`)
	esc = strings.ReplaceAll(esc, `"`, `\"`)
	script := fmt.Sprintf(`tell application "iTerm2"
  activate
  if (count of windows) = 0 then
    set w to (create window with default profile)
    tell current session of w to write text "%s"
  else
    tell current window
      set t to (create tab with default profile)
      tell current session of t to write text "%s"
    end tell
  end if
end tell`, esc, esc)
	_, err := osa(script)
	return err
}
