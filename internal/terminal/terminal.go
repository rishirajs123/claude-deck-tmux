// Package terminal drives iTerm2 via AppleScript to focus, resume, or start
// Claude Code sessions. macOS + iTerm2 only for now (behind these functions so
// other emulators can be added later).
package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tachodril/claude-deck/internal/live"
)

func osa(script string) (string, error) { return osaArgs(script) }

func osaArgs(script string, args ...string) (string, error) {
	out, err := exec.Command("osascript", append([]string{"-e", script}, args...)...).CombinedOutput()
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

// NewSession opens a new iTerm2 tab in dir and starts a fresh claude session.
// A non-empty prompt is passed as claude's initial prompt (submitted on start).
func NewSession(dir, prompt string) error {
	d, err := ensureDir(dir)
	if err != nil {
		return err
	}
	cmd := "claude"
	if strings.TrimSpace(prompt) != "" {
		cmd += " " + shQuote(prompt)
	}
	return openTab(shellCmd(d, cmd))
}

// ensureDir expands ~, requires an absolute path, and creates the directory
// (mkdir -p) if it doesn't exist yet.
func ensureDir(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", fmt.Errorf("directory is required")
	}
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			p = filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	if !filepath.IsAbs(p) {
		return "", fmt.Errorf("use an absolute path or ~: %s", p)
	}
	if info, err := os.Stat(p); err == nil {
		if !info.IsDir() {
			return "", fmt.Errorf("not a directory: %s", p)
		}
		return p, nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("cannot access %s: %v", p, err)
	}
	if err := os.MkdirAll(p, 0o755); err != nil {
		return "", fmt.Errorf("could not create %s: %v", p, err)
	}
	return p, nil
}

// SendText types text into the running session's iTerm2 tab and submits it
// (a trailing newline). Used to inject slash commands like /model, /compact.
func SendText(cwd, text string) error {
	if text == "" {
		return fmt.Errorf("empty command")
	}
	tty := live.TtyForCwd(cwd)
	if tty == "" {
		return fmt.Errorf("no running session in %s", cwd)
	}
	esc := strings.ReplaceAll(text, `\`, `\\`)
	esc = strings.ReplaceAll(esc, `"`, `\"`)
	script := fmt.Sprintf(`tell application "iTerm2"
  repeat with w in windows
    repeat with t in tabs of w
      repeat with s in sessions of t
        if tty of s is %q then
          tell s to write text "%s"
          return "ok"
        end if
      end repeat
    end repeat
  end repeat
  return "notfound"
end tell`, tty, esc)
	out, err := osa(script)
	if err != nil {
		return err
	}
	if out == "notfound" {
		return fmt.Errorf("session tty %s not found", tty)
	}
	return nil
}

// SendMessage pastes text into the running session using bracketed paste (so a
// multi-line message stays a single prompt), then submits it with a newline.
// Text is passed as an argv item, so no AppleScript-string escaping is needed.
func SendMessage(cwd, text string) error {
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("empty message")
	}
	tty := live.TtyForCwd(cwd)
	if tty == "" {
		return fmt.Errorf("no running session in %s", cwd)
	}
	script := `on run argv
  set theText to item 1 of argv
  set theTty to item 2 of argv
  tell application "iTerm2"
    repeat with w in windows
      repeat with t in tabs of w
        repeat with s in sessions of t
          if tty of s is theTty then
            tell s to write text ((ASCII character 27) & "[200~" & theText & (ASCII character 27) & "[201~") newline no
            delay 0.1
            tell s to write text (ASCII character 13) newline no
            return "ok"
          end if
        end repeat
      end repeat
    end repeat
  end tell
  return "notfound"
end run`
	out, err := osaArgs(script, text, tty)
	if err != nil {
		return err
	}
	if out == "notfound" {
		return fmt.Errorf("session tty %s not found", tty)
	}
	return nil
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
	script := `on run argv
  set theCmd to item 1 of argv
  tell application "iTerm2"
    activate
    if (count of windows) = 0 then
      set w to (create window with default profile)
      tell current session of w to write text theCmd
    else
      tell current window
        set t to (create tab with default profile)
        tell current session of t to write text theCmd
      end tell
    end if
  end tell
end run`
	_, err := osaArgs(script, cmd)
	return err
}
