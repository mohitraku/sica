package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/mohitraku/sica/internal/config"
	"github.com/mohitraku/sica/internal/storage"
	"github.com/mohitraku/sica/internal/tui"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("sica %s (commit %s, built %s)\n", Version, Commit, Date)
		return
	}

	dataDir, source := sicaDir()

	db, err := storage.Open(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	hStore := storage.NewHabitStore(db)
	eStore := storage.NewEntryStore(db)

	model := tui.New(hStore, eStore, dataDir, source)
	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func sicaDir() (dir, source string) {
	if dir := os.Getenv("SICA_DATA_DIR"); dir != "" {
		return dir, "env"
	}
	cfg, _ := config.Load()
	if cfg.DataDir != "" {
		return cfg.DataDir, "config"
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".sica", "default"
	}
	return home + "/.sica", "default"
}
