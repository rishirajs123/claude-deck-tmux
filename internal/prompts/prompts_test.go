package prompts

import "testing"

func TestDetect(t *testing.T) {
	edit := `Some earlier output here
tool ran and printed things

╭──────────────────────────────────────────────╮
│ Do you want to make this edit to app.ts?       │
│ ❯ 1. Yes                                        │
│   2. Yes, and don't ask again this session      │
│   3. No, and tell Claude what to do (esc)        │
╰──────────────────────────────────────────────╯

 > `
	p := Detect(edit)
	if p == nil {
		t.Fatal("expected a prompt, got nil")
	}
	if len(p.Options) != 3 || p.Options[0].Key != "1" || p.Options[2].Key != "3" {
		t.Fatalf("bad options: %+v", p.Options)
	}
	if p.Question == "" {
		t.Fatal("expected a question")
	}

	bash := `╭────────────────────────────────────────╮
│ Bash command                            │
│   rm -rf ./build                        │
│ Do you want to proceed?                 │
│ ❯ 1. Yes                                 │
│   2. No, and tell Claude what to do      │
╰────────────────────────────────────────╯
 > `
	if p := Detect(bash); p == nil || len(p.Options) != 2 {
		t.Fatalf("bash prompt not detected: %+v", p)
	}

	// Not a prompt: normal idle screen.
	if p := Detect("assistant finished.\n\nHere is the summary of changes.\n\n > "); p != nil {
		t.Fatalf("false positive on idle screen: %+v", p)
	}
	// Not a prompt: a numbered list in output (no yes/no/allow keywords).
	if p := Detect("Steps:\n1. Clone the repo\n2. Run make\n3. Open the app\n\n > "); p != nil {
		t.Fatalf("false positive on numbered list: %+v", p)
	}
}
