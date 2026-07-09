// Package prompts detects when a Claude Code session is sitting on a
// permission/confirmation prompt, by pattern-matching the tail of its terminal
// screen. Best-effort: the TUI format can change across Claude Code versions,
// so detection is conservative (it prefers missing a prompt over false alarms).
package prompts

import (
	"regexp"
	"strings"

	"github.com/tachodril/claude-deck/internal/model"
)

var optRe = regexp.MustCompile(`^([0-9])[.)]\s+(\S.*)$`)

// promptKeywords: at least one must appear in the question or an option label
// for a numbered block to count as a permission prompt (not just any list).
var promptKeywords = []string{"yes", "no", "proceed", "allow", "don't ask", "do you want", "approve", "reject"}

// Detect returns the prompt a session is waiting on, or nil if it doesn't look
// like one. screen is the tail of the visible terminal buffer.
func Detect(screen string) *model.Prompt {
	raw := strings.Split(screen, "\n")
	last := len(raw)
	for last > 0 && clean(raw[last-1]) == "" {
		last--
	}
	if last == 0 {
		return nil
	}
	from := last - 18
	if from < 0 {
		from = 0
	}

	var opts []model.PromptOption
	firstOpt, lastOpt := -1, -1
	for i := from; i < last; i++ {
		m := optRe.FindStringSubmatch(clean(raw[i]))
		if m == nil {
			continue
		}
		if firstOpt < 0 {
			firstOpt = i
		}
		lastOpt = i
		opts = append(opts, model.PromptOption{Key: m[1], Label: m[2]})
	}
	// Need a couple of options, and they must sit near the bottom (an answered
	// prompt would have scrolled up with tool output beneath it).
	if len(opts) < 2 || last-lastOpt > 9 {
		return nil
	}

	question := ""
	for i := firstOpt - 1; i >= 0 && i >= firstOpt-6; i-- {
		c := clean(raw[i])
		if c == "" {
			continue
		}
		lc := strings.ToLower(c)
		if strings.Contains(c, "?") || strings.Contains(lc, "do you want") || strings.HasPrefix(lc, "allow") {
			question = c
			break
		}
	}
	if !looksLikePrompt(question, opts) {
		return nil
	}
	return &model.Prompt{Question: question, Options: opts}
}

func looksLikePrompt(q string, opts []model.PromptOption) bool {
	blob := strings.ToLower(q)
	for _, o := range opts {
		blob += " " + strings.ToLower(o.Label)
	}
	for _, kw := range promptKeywords {
		if strings.Contains(blob, kw) {
			return true
		}
	}
	return false
}

// clean strips box-drawing / pointer glyphs and surrounding whitespace so an
// option line like "│ ❯ 1. Yes │" becomes "1. Yes".
func clean(s string) string {
	s = strings.Map(func(r rune) rune {
		switch r {
		case '│', '╭', '╮', '╰', '╯', '─', '╌', '┃', '┆', '┊', '❯', '›', '»', '▶', '▌', '▐', '•':
			return ' '
		}
		return r
	}, s)
	return strings.TrimSpace(s)
}
