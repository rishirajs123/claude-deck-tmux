package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/tachodril/claude-deck/internal/ingest"
	"github.com/tachodril/claude-deck/internal/server"
	"github.com/tachodril/claude-deck/internal/store"
)

// version is overridable at build time via -ldflags "-X main.version=v1.2.3".
var version = "dev"

func main() {
	port := flag.String("port", "7420", "port to serve on")
	noOpen := flag.Bool("no-open", false, "do not auto-open the browser")
	showVer := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *showVer {
		fmt.Println("claude-deck", version)
		return
	}

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	claudeDir := filepath.Join(home, ".claude")

	st, err := store.Open(filepath.Join(claudeDir, "claude-deck.db"))
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer st.Close()

	log.Println("ingesting sessions from", claudeDir, "…")
	n, err := ingest.Run(claudeDir, st)
	if err != nil {
		log.Printf("ingest warning: %v", err)
	}
	log.Printf("ingested %d sessions", n)

	url := "http://localhost:" + *port
	log.Printf("ClaudeDeck %s → %s", version, url)
	// Auto-open the browser only for interactive runs (not under launchd/pipes).
	if !*noOpen && interactive() {
		go func() {
			time.Sleep(500 * time.Millisecond)
			_ = exec.Command("open", url).Start()
		}()
	}
	if err := server.New(st).Listen(":" + *port); err != nil {
		log.Fatal(err)
	}
}

func interactive() bool {
	fi, err := os.Stdout.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}
